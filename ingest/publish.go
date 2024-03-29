package ingest

import (
	"context"
	"fmt"
	"io"
	"sync/atomic"
	"time"

	"eaglesong.dev/gunk/internal"
	"eaglesong.dev/gunk/model"
	"eaglesong.dev/gunk/sinks/grabber"
	"eaglesong.dev/gunk/transcode/opus"
	"eaglesong.dev/hls"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/av/pubsub"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

const webExpiry = 60 * time.Second

func (m *Manager) Publish(ctx context.Context, auth model.ChannelAuth, src av.Demuxer) error {
	name := auth.Name
	l := zerolog.Ctx(ctx)
	l.UpdateContext(func(c zerolog.Context) zerolog.Context {
		c = c.Str("channel", auth.Name)
		c = c.Str("user_id", auth.UserID)
		return c
	})
	streams, err := src.Streams()
	if err != nil {
		return fmt.Errorf("reading streams: %w", err)
	}
	// log stream configuration
	for i, cd := range streams {
		ev := l.Debug().Int("idx", i).Stringer("codec", cd.Type())
		if ev.Enabled() {
			internal.CodecTag(cd, ev)
			ev.Send()
		}
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
		return fmt.Errorf("setting up frame grabber: %w", err)
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
	ch.name = name
	p := ch.setStream(q, aacq, opusq, m.WorkDir, m.PublishMode)
	defer func() {
		l.Info().Msg("stopped publishing")
		if ch.stopStream(q) && m.PublishEvent != nil {
			m.PublishEvent(auth, false, grabber.Result{})
		}
	}()
	// announce
	l.Info().Msg("started publishing")
	if m.PublishEvent != nil {
		m.PublishEvent(auth, true, grabber.Result{})
	}
	// start outputs
	eg.Go(func() error {
		err := ch.copyWeb(p, q.Latest())
		if err != nil {
			err = fmt.Errorf("web publish: %w", err)
		}
		return err
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

func (ch *channel) setStream(q, aacq, opusq *pubsub.Queue, workDir string, mode hls.Mode) *hls.Publisher {
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
	ch.web = &hls.Publisher{
		WorkDir: workDir,
		Mode:    mode,
	}
	ch.stoppedAt = time.Time{}
	atomic.StoreUintptr(&ch.live, uintptr(statePending))
	return ch.web
}

func (ch *channel) stopStream(q *pubsub.Queue) bool {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	if ch.ingest != q {
		return false
	}
	atomic.StoreUintptr(&ch.live, uintptr(stateOffline))
	ch.ingest = nil
	ch.aac = nil
	ch.opus = nil
	ch.stoppedAt = time.Now()
	return true
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
	needKeys := 3
	log.Info().Str("channel", ch.name).Msgf("live in %d", needKeys)
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
		if pkt.IsKeyFrame && needKeys > 0 {
			needKeys--
			ev := log.Info().Str("channel", ch.name)
			if needKeys == 0 {
				ev.Msgf("going live")
				atomic.StoreUintptr(&ch.live, uintptr(stateLive))
			} else {
				ev.Msgf("live in %d", needKeys)
			}
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
		err := opus.Convert(q.Latest(), ret, bitrate)
		if err != nil {
			err = fmt.Errorf("opus conversion: %w", err)
		}
		return err
	})
	return ret
}
