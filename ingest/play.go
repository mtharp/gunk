package ingest

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"sync/atomic"

	"eaglesong.dev/gunk/model"
	"eaglesong.dev/gunk/sinks/playrtc"
	"eaglesong.dev/gunk/sinks/rtsp"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/format/ts"
	"github.com/pkg/errors"
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

func (m *Manager) ServeHLS(rw http.ResponseWriter, req *http.Request, name string) error {
	ch := m.channel(name)
	if ch == nil {
		return ErrNoChannel
	}
	host, _, _ := net.SplitHostPort(req.RemoteAddr)
	if host == "" {
		host = req.RemoteAddr
	}
	if host != "" {
		ch.hlsViewed(host)
	}
	p := ch.getHLS()
	if p == nil {
		return ErrNoChannel
	}
	p.ServeHTTP(rw, req)
	return nil
}

func (m *Manager) ServeSDP(rw http.ResponseWriter, req *http.Request, name string) error {
	ch := m.channel(name)
	src := ch.queue(true)
	if src == nil {
		return ErrNoChannel
	}
	return playrtc.HandleSDP(rw, req, src, func(delta int) { ch.addViewer(int32(delta)) })
}

func (m *Manager) GetRTSPSource(req *rtsp.Request) (av.Demuxer, error) {
	chname := req.URL.Path
	if strings.HasPrefix(chname, "/") {
		chname = chname[1:]
	}
	chname = strings.Split(chname, "/")[0]
	src := m.channel(chname).queue(true)
	if src == nil {
		return nil, rtsp.ErrNotFound
	}
	// TODO: viewer count
	return src, nil
}

func (m *Manager) PopulateLive(infos []*model.ChannelInfo) {
	for _, info := range infos {
		ch := m.channel(info.Name)
		if ch == nil {
			continue
		}
		info.Live = ch.isLive()
		info.Viewers = ch.currentViewers()
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
