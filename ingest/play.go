package ingest

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"sync/atomic"

	"eaglesong.dev/gunk/model"
	"eaglesong.dev/gunk/sinks/playrtc"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/format/ts"
	"github.com/rs/zerolog"
)

var ErrNoChannel = errors.New("channel not found")

func (m *Manager) ServeTS(rw http.ResponseWriter, req *http.Request, name string) error {
	ch := m.channel(name)
	src := ch.queue(false)
	if src == nil {
		return ErrNoChannel
	}
	rw.Header().Set("Content-Type", "video/MP2T")
	rw.Header().Set("Transfer-Encoding", "chunked")
	muxer := ts.NewMuxer(rw)
	streams, _ := src.Streams()
	muxer.WriteHeader(streams)
	ch.addViewer(1)
	defer ch.addViewer(-1)
	return copyStream(req.Context(), muxer, src)
}

func (m *Manager) ServeWeb(rw http.ResponseWriter, req *http.Request, name string) error {
	if req.Header.Get("Origin") != "" {
		rw.Header().Set("Access-Control-Allow-Origin", "*")
	}
	ch := m.channel(name)
	if ch == nil {
		return ErrNoChannel
	}
	host, _, _ := net.SplitHostPort(req.RemoteAddr)
	if host == "" {
		host = req.RemoteAddr
	}
	if host != "" {
		ch.webViewed(host)
	}
	p := ch.getWeb()
	if p == nil {
		return ErrNoChannel
	}
	p.ServeHTTP(rw, req)
	return nil
}

func (m *Manager) OfferSDP(ctx context.Context, name string, sendCandidate playrtc.CandidateSender) (*playrtc.Sender, error) {
	ch := m.channel(name)
	if ch == nil {
		return nil, ErrNoChannel
	}
	src := ch.queue(true)
	if src == nil {
		return nil, ErrNoChannel
	}
	addViewer := func(delta int) { ch.addViewer(int32(delta)) }
	l := zerolog.Ctx(ctx).With().Str("channel", name).Logger()
	ctx = l.WithContext(ctx)
	return m.rtc.OfferToSend(ctx, src, addViewer, sendCandidate)
}

func (m *Manager) PopulateLive(infos []*model.ChannelInfo) {
	for _, info := range infos {
		ch := m.channel(info.Name)
		if ch == nil {
			continue
		}
		info.Live = ch.isLive()
		info.Viewers = ch.currentViewers()
		if m.UseDASH {
			info.WebURL = ch.getWeb().MPD()
		} else {
			info.WebURL = ch.getWeb().Playlist()
		}
		info.NativeURL = ch.getWeb().Playlist()
		info.RTC = atomic.LoadUintptr(&ch.rtc) != 0
	}
}

func copyStream(ctx context.Context, dest av.Muxer, src av.Demuxer) error {
	for ctx.Err() == nil {
		pkt, err := src.ReadPacket()
		if err == io.EOF || ctx.Err() != nil {
			return nil
		} else if err != nil {
			return err
		}
		if err := dest.WritePacket(pkt); err != nil {
			return err
		}
	}
	return nil
}
