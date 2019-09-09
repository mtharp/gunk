package ingest

import (
	"sync"
	"sync/atomic"
	"time"

	"eaglesong.dev/gunk/ingest/ftl"
	"eaglesong.dev/gunk/model"
	"eaglesong.dev/gunk/sinks/grabber"
	"eaglesong.dev/hls"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/av/pubsub"
)

type PublishEvent func(auth model.ChannelAuth, live bool, thumb grabber.Result)

type Manager struct {
	OpusBitrate  int
	PublishEvent PublishEvent
	FTL          ftl.Server

	channels sync.Map
}

func (m *Manager) Initialize() {
	m.FTL.Publish = m.Publish
}

type channel struct {
	mu        sync.Mutex
	ingest    *pubsub.Queue
	aac, opus *pubsub.Queue
	hls       *hls.Publisher
	stoppedAt time.Time

	live, rtc uintptr
	viewers   int32 // excluding hls
	refPos    int64
}

func (m *Manager) channel(name string) *channel {
	v, _ := m.channels.Load(name)
	if v != nil {
		return v.(*channel)
	}
	return nil
}

func (ch *channel) queue(opus bool) av.Demuxer {
	if ch == nil {
		return nil
	}
	ch.mu.Lock()
	defer ch.mu.Unlock()
	var q *pubsub.Queue
	if opus {
		q = ch.opus
	} else {
		q = ch.aac
	}
	if q == nil {
		return nil
	}
	return q.Latest()
}

func (ch *channel) isLive() bool {
	if ch == nil {
		return false
	}
	return atomic.LoadUintptr(&ch.live) != 0
}

func (ch *channel) addViewer(delta int32) {
	if ch == nil {
		return
	}
	atomic.AddInt32(&ch.viewers, delta)
}

func (ch *channel) getHLS() *hls.Publisher {
	if ch == nil {
		return nil
	}
	ch.mu.Lock()
	p := ch.hls
	ch.mu.Unlock()
	return p
}

func (ch *channel) currentViewers() int {
	v := int(atomic.LoadInt32(&ch.viewers))
	if p := ch.getHLS(); p != nil {
		v += p.Viewers()
	}
	return v
}
