package ingest

import (
	"sync"
	"sync/atomic"
	"time"

	"eaglesong.dev/gunk/ingest/whip"
	"eaglesong.dev/gunk/internal/rtcengine"
	"eaglesong.dev/gunk/model"
	"eaglesong.dev/gunk/sinks/grabber"
	"eaglesong.dev/hls"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/av/pubsub"
)

const webViewTimeout = 16 * time.Second

type PublishEvent func(auth model.ChannelAuth, live bool, thumb grabber.Result)

type Manager struct {
	OpusBitrate  int
	PublishEvent PublishEvent
	PublishMode  hls.Mode
	WorkDir      string
	RTCHost      string
	RTCWindow    time.Duration

	channels sync.Map
	rtc      *rtcengine.Engine
}

func (m *Manager) Initialize() error {
	var err error
	m.rtc, err = rtcengine.New(m.RTCHost)
	if err != nil {
		return err
	}
	m.rtc.ReceiveWindow = m.RTCWindow
	return nil
}

type channel struct {
	mu        sync.Mutex
	ingest    *pubsub.Queue
	aac, opus *pubsub.Queue
	web       *hls.Publisher

	whip   *whip.Receiver
	whipID string

	stoppedAt time.Time
	name      string

	live, rtc uintptr
	viewers   int32 // excluding web
	webv      sync.Map
	webvTotal int32
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

type liveState uintptr

const (
	stateOffline liveState = iota
	statePending
	stateLive
)

func (ch *channel) isLive() liveState {
	if ch == nil {
		return stateOffline
	}
	return liveState(atomic.LoadUintptr(&ch.live))
}

func (ch *channel) addViewer(delta int32) {
	if ch == nil {
		return
	}
	atomic.AddInt32(&ch.viewers, delta)
}

func (ch *channel) webViewed(host string) {
	ch.webv.Store(host, time.Now())
}

func (ch *channel) countWebViewers() {
	var views int32
	ch.webv.Range(func(key, value interface{}) bool {
		t := value.(time.Time)
		if time.Since(t) > webViewTimeout {
			ch.webv.Delete(key)
		} else {
			views++
		}
		return true
	})
	atomic.StoreInt32(&ch.webvTotal, views)
}

func (ch *channel) getWeb() *hls.Publisher {
	if ch == nil {
		return nil
	}
	ch.mu.Lock()
	p := ch.web
	ch.mu.Unlock()
	return p
}

func (ch *channel) currentViewers() int {
	v := int(atomic.LoadInt32(&ch.viewers))
	v += int(atomic.LoadInt32(&ch.webvTotal))
	return v
}
