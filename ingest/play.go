package ingest

import (
	"net/http"
	"strings"

	"eaglesong.dev/gunk/model"
	"eaglesong.dev/gunk/sinks/playrtc"
	"eaglesong.dev/gunk/sinks/rtsp"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/format/ts"
	"github.com/pkg/errors"
)

var ErrNoChannel = errors.New("channel not found")

func (m *Manager) queue(name string, opus bool) av.Demuxer {
	m.mu.Lock()
	defer m.mu.Unlock()
	if opus {
		return m.opus[name].Latest()
	}
	return m.ingest[name].Latest()
}

func (m *Manager) ServeTS(rw http.ResponseWriter, req *http.Request, name string) error {
	src := m.queue(name, false)
	if src == nil {
		return ErrNoChannel
	}
	rw.Header().Set("Content-Type", "video/MP2T")
	rw.Header().Set("Transfer-Encoding", "chunked")
	muxer := ts.NewMuxer(rw)
	streams, _ := src.Streams()
	muxer.WriteHeader(streams)
	return copyStream(req.Context(), muxer, src)
}

func (m *Manager) ServeHLS(rw http.ResponseWriter, req *http.Request, name string) error {
	m.mu.Lock()
	hls := m.hls[name]
	m.mu.Unlock()
	if hls == nil {
		return ErrNoChannel
	}
	hls.ServeHTTP(rw, req)
	return nil
}

func (m *Manager) ServeSDP(rw http.ResponseWriter, req *http.Request, name string) error {
	src := m.queue(name, true)
	if src == nil {
		return ErrNoChannel
	}
	return playrtc.HandleSDP(rw, req, src)
}

func (m *Manager) GetRTSPSource(req *rtsp.Request) (av.Demuxer, error) {
	chname := req.URL.Path
	if strings.HasPrefix(chname, "/") {
		chname = chname[1:]
	}
	chname = strings.Split(chname, "/")[0]
	src := m.queue(chname, true)
	if src == nil {
		return nil, rtsp.ErrNotFound
	}
	return src, nil
}

func (m *Manager) PopulateLive(infos []*model.ChannelInfo) {
	m.mu.Lock()
	for _, info := range infos {
		if m.ingest[info.Name] != nil {
			info.Live = true
		}
	}
	m.mu.Unlock()
}
