package web

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"eaglesong.dev/gunk/model"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/hlog"
)

func (s *Server) listChannels(ctx context.Context, m channelMarkers) ([]*model.ChannelInfo, error) {
	infos, err := model.ListChannelInfo(ctx)
	if err != nil {
		return nil, err
	}
	s.Channels.PopulateLive(infos)
	if m != nil {
		infos = m.Filter(infos)
	}
	for _, info := range infos {
		s.populateChannel(info)
	}
	return infos, nil
}

func (s *Server) populateChannel(info *model.ChannelInfo) {
	u, _ := s.router.Get("thumbs").URL("channel", info.Name, "timestamp", strconv.FormatInt(info.Last, 10))
	info.Thumb = u.String()
	liveU, _ := s.router.Get("live").URL("channel", info.Name)
	if s.AdvertiseLive != nil {
		liveU = s.AdvertiseLive.ResolveReference(liveU)
	}
	info.LiveURL = liveU.String()
	if info.WebURL != "" {
		webU, _ := s.router.Get("web").URL("channel", info.Name, "filename", info.WebURL)
		nativeU, _ := s.router.Get("web").URL("channel", info.Name, "filename", info.NativeURL)
		if s.HLSBase != nil {
			webU = s.HLSBase.ResolveReference(webU)
			nativeU = s.HLSBase.ResolveReference(nativeU)
		}
		info.WebURL = webU.String()
		info.NativeURL = nativeU.String()
	}
}

func (s *Server) viewChannelInfo(rw http.ResponseWriter, req *http.Request) {
	infos, err := s.listChannels(req.Context(), nil)
	if err != nil {
		hlog.FromRequest(req).Err(err).Msg("failed listing channels")
		http.Error(rw, "", 500)
	}
	ret := struct {
		Time     int64                         `json:"time"`
		Channels map[string]*model.ChannelInfo `json:"channels"`
		Recent   []string                      `json:"recent"`
	}{
		Time:     time.Now().UnixNano() / 1000000,
		Channels: make(map[string]*model.ChannelInfo),
	}
	for _, info := range infos {
		ret.Channels[info.Name] = info
		ret.Recent = append(ret.Recent, info.Name)
	}
	if req.Header.Get("Origin") != "" {
		rw.Header().Set("Access-Control-Allow-Origin", "*")
	}
	writeJSON(rw, ret)
}

func (s *Server) viewThumb(rw http.ResponseWriter, req *http.Request) {
	chname := mux.Vars(req)["channel"]
	jpeg, err := model.GetThumb(req.Context(), chname)
	if err == pgx.ErrNoRows {
		hlog.FromRequest(req).Info().Str("channel", chname).Msg("channel not found")
		http.NotFound(rw, req)
		return
	} else if err != nil {
		hlog.FromRequest(req).Err(err).Str("channel", chname).Msg("failed to get thumbnail")
		http.Error(rw, "", 500)
		return
	}
	rw.Header().Set("Cache-Control", "max-age=86400, public, immutable")
	rw.Header().Set("Content-Type", "image/jpeg")
	rw.Write(jpeg)
}

func (s *Server) viewPlaylist(rw http.ResponseWriter, req *http.Request) {
	chname := mux.Vars(req)["channel"]
	liveU, _ := s.router.Get("live").URL("channel", chname)
	if s.AdvertiseLive != nil {
		liveU = s.AdvertiseLive.ResolveReference(liveU)
	}
	rw.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	fmt.Fprintln(rw, liveU)
}
