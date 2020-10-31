package web

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"eaglesong.dev/gunk/model"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx"
)

func (s *Server) listChannels() ([]*model.ChannelInfo, error) {
	infos, err := model.ListChannelInfo()
	if err != nil {
		return nil, err
	}
	for _, info := range infos {
		s.populateChannel(info)
	}
	s.Channels.PopulateLive(infos)
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
}

func (s *Server) viewChannelInfo(rw http.ResponseWriter, req *http.Request) {
	infos, err := s.listChannels()
	if err != nil {
		log.Printf("error: listing channels: %s", err)
		http.Error(rw, "", 500)
	}
	ret := struct {
		Time     int64                         `json:"time"`
		Channels map[string]*model.ChannelInfo `json:"channels"`
		Recent   []string                      `json:"recent"`
		BaseURL  string                        `json:"base_url"`
	}{
		Time:     time.Now().UnixNano() / 1000000,
		Channels: make(map[string]*model.ChannelInfo),
	}
	for _, info := range infos {
		ret.Channels[info.Name] = info
		ret.Recent = append(ret.Recent, info.Name)
	}
	if s.HLSBase != "" {
		ret.BaseURL = strings.TrimSuffix(s.HLSBase, "/")
	}
	rw.Header().Set("Access-Control-Allow-Origin", "*")
	writeJSON(rw, ret)
}

func (s *Server) viewThumb(rw http.ResponseWriter, req *http.Request) {
	chname := mux.Vars(req)["channel"]
	jpeg, err := model.GetThumb(chname)
	if err == pgx.ErrNoRows {
		log.Printf("not found: %s", req.URL)
		http.NotFound(rw, req)
		return
	} else if err != nil {
		log.Printf("error: getting thumbnail: %s", err)
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
