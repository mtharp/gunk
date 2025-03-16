package server

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"eaglesong.dev/gunk/internal/model"
	"github.com/rs/zerolog/hlog"
)

type channelEntry struct {
	model.ChannelInfo
	Live  bool   `json:"live"`
	Thumb string `json:"thumb"`

	HlsURL      string `json:"hls_url"`
	PlaylistURL string `json:"playlist_url"`
	LiveURL     string `json:"live_url"`
	WhepURL     string `json:"whep_url"`
	Viewers     int    `json:"viewers"`
}

type channelStatus struct {
	Name    string
	Viewers int
}

func (s *Server) getLivePaths(ctx context.Context) (map[string]channelStatus, error) {
	pathList, err := s.MtxAPI.ListPaths(ctx)
	if err != nil {
		return nil, err
	}
	livePaths := make(map[string]channelStatus)
	for _, p := range pathList {
		if !p.Ready {
			continue
		}
		hlsViews := s.hlsViews.Count(p.Name)
		livePaths[p.Name] = channelStatus{
			Name:    p.Name,
			Viewers: p.Viewers() + hlsViews,
		}
	}
	return livePaths, nil
}

func (s *Server) viewChannels(rw http.ResponseWriter, req *http.Request) {
	// identify which paths are live
	livePaths, err := s.getLivePaths(req.Context())
	if err != nil {
		hlog.FromRequest(req).Err(err).Msg("failed to list live paths")
		http.Error(rw, "", http.StatusInternalServerError)
		return
	}
	// list everything from the database and fill in the live info
	modelChannels, err := s.Model.ListRecentChannels(req.Context())
	if err != nil {
		hlog.FromRequest(req).Err(err).Msg("failed to list channel definitions")
		http.Error(rw, "", http.StatusInternalServerError)
		return
	}
	byName := make(map[string]channelEntry)
	var recent []string
	for _, ch := range modelChannels {
		status := livePaths[ch.Name]
		escaped := url.PathEscape(ch.Name)
		byName[ch.Name] = channelEntry{
			ChannelInfo: *ch,
			Live:        status.Name != "",
			Viewers:     status.Viewers,
			Thumb:       fmt.Sprintf("/thumbs/%s/%d.jpg", escaped, ch.Last),
			WhepURL:     fmt.Sprintf("%s/whep/%s", s.BaseURL, escaped),
			HlsURL:      fmt.Sprintf("%s/hls/%s/index.m3u8", s.AdvertiseHLS, escaped),
			LiveURL:     fmt.Sprintf("%s/%s", s.AdvertiseRTSP, escaped),
			PlaylistURL: "/live/" + escaped,
		}
		recent = append(recent, ch.Name)
	}
	ret := map[string]any{
		"time":     time.Now().UnixNano() / 1000000,
		"channels": byName,
		"recent":   recent,
	}
	if req.Header.Get("Origin") != "" {
		rw.Header().Set("Access-Control-Allow-Origin", "*")
	}
	writeJSON(rw, ret)
}
