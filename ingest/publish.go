package ingest

import (
	"context"
	"io"
	"log"
	"sync/atomic"
	"time"

	"eaglesong.dev/gunk/model"
	"eaglesong.dev/gunk/sinks/grabber"
	"eaglesong.dev/gunk/transcode/opus"
	"eaglesong.dev/hls/dash"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/av/pubsub"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

const webExpiry = 60 * time.Second

func (m *Manager) Publish(auth model.ChannelAuth, kind, remote string, src av.Demuxer) error {
	name := auth.Name
	streams, err := src.Streams()
	if err != nil {
		return errors.Wrap(err, "reading streams")
	}
	q := pubsub.NewQueue()
	q.WriteHeader(streams)
	eg, ctx := errgroup.WithContext(context.Background())
	go func() {
		<-ctx.Done()
		q.Close()
	}()
	// grab keyframes for thumbnail
	grabch, err := grabber.Grab(name, q.Oldest())
	if err != nil {
		return errors.Wrap(err, "setting up frame grabber")
	}
	aacq := q
	opusq := q
	switch audioType(streams) {
	case av.OPUS, 0:
	default:
		opusq = convertOpus(eg, q, m.OpusBitrate)
	}

	// go live
	v, _ := m.channels.LoadOrStore(name, new(channel))
	ch := v.(*channel)
	p := ch.setStream(q, aacq, opusq, m.WorkDir)
	defer func() {
		log.Printf("[%s] publish of %s stopped", kind, auth.Name)
		ch.stopStream(q)
		if m.PublishEvent != nil {
			m.PublishEvent(auth, false, grabber.Result{})
		}
	}()
	// announce
	log.Printf("[%s] user %s started publishing to %s from %s", kind, auth.UserID, auth.Name, remote)
	if m.PublishEvent != nil {
		m.PublishEvent(auth, true, grabber.Result{})
	}
	// start outputs
	eg.Go(func() error {
		return errors.Wrap(ch.copyWeb(p, q.Oldest()), "web publish")
	})
	eg.Go(func() error {
		// notify ws clients when thumbnail is updated
		for thumb := range grabch {
			ch.countWebViewers()
			if m.PublishEvent != nil {
				m.PublishEvent(auth, true, thumb)
			}
			if thumb.HasBframes {
				atomic.StoreUintptr(&ch.rtc, 0)
			} else {
				atomic.StoreUintptr(&ch.rtc, 1)
			}
		}
		return nil
	})
	// copy
	eg.Go(func() error { return ch.copyStream(ctx, q, src) })
	return eg.Wait()
}

func (m *Manager) Cleanup() {
	m.channels.Range(func(k, v interface{}) bool {
		v.(*channel).cleanup()
		return true
	})
}

func (ch *channel) setStream(q, aacq, opusq *pubsub.Queue, workDir string) *dash.Publisher {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	if ch.ingest != nil {
		ch.ingest.Close()
	}
	ch.ingest = q
	ch.aac = aacq
	ch.opus = opusq
	if ch.web != nil {
		ch.web.Close()
	}
	ch.web = &dash.Publisher{
		WorkDir: workDir,
	}
	ch.stoppedAt = time.Time{}
	atomic.StoreUintptr(&ch.live, 1)
	return ch.web
}

func (ch *channel) stopStream(q *pubsub.Queue) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	if ch.ingest != q {
		return
	}
	atomic.StoreUintptr(&ch.live, 0)
	ch.ingest = nil
	ch.aac = nil
	ch.opus = nil
	ch.stoppedAt = time.Now()
}

func (ch *channel) copyStream(ctx context.Context, dest *pubsub.Queue, src av.Demuxer) error {
	defer dest.Close()
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		pkt, err := src.ReadPacket()
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}
		if err := dest.WritePacket(pkt); err != nil {
			return err
		}
	}
}

func (ch *channel) copyWeb(dest av.Muxer, src av.Demuxer) error {
	var streams []av.CodecData
	var err error
	if streams, err = src.Streams(); err != nil {
		return err
	}
	if err = dest.WriteHeader(streams); err != nil {
		return err
	}
	for {
		pkt, err := src.ReadPacket()
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}
		if err := dest.WritePacket(pkt); err != nil {
			return err
		}
	}
}

func (ch *channel) cleanup() {
	ch.mu.Lock()
	if ch.web != nil && !ch.stoppedAt.IsZero() && time.Since(ch.stoppedAt) > webExpiry {
		ch.web.Close()
		ch.web = nil
	}
	ch.mu.Unlock()
}

func audioType(streams []av.CodecData) av.CodecType {
	for _, stream := range streams {
		if stream.Type().IsAudio() {
			return stream.Type()
		}
	}
	return 0
}

func convertOpus(eg *errgroup.Group, q *pubsub.Queue, bitrate int) *pubsub.Queue {
	if bitrate == 0 {
		bitrate = 128000
	}
	ret := pubsub.NewQueue()
	eg.Go(func() error {
		defer ret.Close()
		return errors.Wrap(opus.Convert(q.Oldest(), ret, bitrate), "opus conversion")
	})
	return ret
}
