package web

import (
	"log"
	"net/http"
	"strconv"

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
	rw.Header().Set("Access-Control-Allow-Origin", "*")
	writeJSON(rw, infos)
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
