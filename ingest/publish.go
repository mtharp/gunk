package ingest

import (
	"context"
	"io"
	"log"
	"sync"
	"time"

	"eaglesong.dev/gunk/ingest/ftl"
	"eaglesong.dev/gunk/model"
	"eaglesong.dev/gunk/sinks/grabber"
	"eaglesong.dev/gunk/transcode/opus"
	"eaglesong.dev/hls"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/av/avutil"
	"github.com/nareix/joy4/av/pubsub"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

type Manager struct {
	OpusBitrate  int
	PublishEvent PublishEvent
	FTL          ftl.Server

	mu     sync.Mutex
	ingest map[string]*pubsub.Queue
	opus   map[string]*pubsub.Queue
	hls    map[string]*hls.Publisher
}

func (m *Manager) Initialize() {
	m.FTL.Publish = m.Publish
}

type PublishEvent func(auth model.ChannelAuth, live bool, thumbTime time.Time)

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
	grabch, err := grabber.Grab(name, q.Latest())
	if err != nil {
		return errors.Wrap(err, "setting up frame grabber")
	}
	opusq := q
	switch audioType(streams) {
	case opus.OPUS, 0:
	default:
		opusq = convertOpus(eg, q, m.OpusBitrate)
	}

	// go live
	m.mu.Lock()
	if m.ingest == nil {
		m.ingest = make(map[string]*pubsub.Queue)
		m.opus = make(map[string]*pubsub.Queue)
		m.hls = make(map[string]*hls.Publisher)
	}
	if existing := m.ingest[name]; existing != nil {
		existing.Close()
	}
	m.ingest[name] = q
	m.opus[name] = opusq
	p := m.hls[name]
	if p != nil {
		// stream restarted so viewer should reset their decoder
		p.Discontinuity()
	} else {
		p = new(hls.Publisher)
		m.hls[name] = p
	}
	m.mu.Unlock()

	defer func() {
		log.Printf("[%s] publish of %s stopped", kind, auth.Name)
		m.mu.Lock()
		if m.ingest[name] == q {
			delete(m.ingest, name)
			delete(m.opus, name)
		}
		m.mu.Unlock()
		if m.PublishEvent != nil {
			m.PublishEvent(auth, false, time.Time{})
		}
	}()
	// announce
	log.Printf("[%s] user %s started publishing to %s from %s", kind, auth.UserID, auth.Name, remote)
	if m.PublishEvent != nil {
		m.PublishEvent(auth, true, time.Time{})
	}
	// start outputs
	eg.Go(func() error {
		return errors.Wrap(avutil.CopyFile(p, q.Latest()), "hls publish")
	})
	eg.Go(func() error {
		defer q.Close()
		return copyStream(ctx, q, src)
	})
	eg.Go(func() error {
		// notify ws clients when thumbnail is updated
		for thumbTime := range grabch {
			if m.PublishEvent != nil {
				m.PublishEvent(auth, true, thumbTime)
			}
		}
		return nil
	})
	return eg.Wait()
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
		return errors.Wrap(opus.Convert(q.Latest(), ret, bitrate), "opus conversion")
	})
	return ret
}

func copyStream(ctx context.Context, dest av.PacketWriter, src av.PacketReader) error {
	for ctx.Err() == nil {
		pkt, err := src.ReadPacket()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		if err := dest.WritePacket(pkt); err != nil {
			return err
		}
	}
	return nil
}
