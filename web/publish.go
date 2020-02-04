package web

import (
	"log"
	"net/http"
	"net/url"

	"eaglesong.dev/gunk/model"
	"github.com/gorilla/mux"
	"github.com/nareix/joy4/format/ts"
)

func (s *Server) viewPublishTS(rw http.ResponseWriter, req *http.Request) {
	chname := mux.Vars(req)["channel"]
	u2 := &url.URL{
		Path:     chname,
		RawQuery: req.URL.RawQuery,
	}
	auth, err := model.VerifyRTMP(u2)
	if err != nil {
		log.Printf("[ts] error: %s from %s: %s", req.URL.Path, req.RemoteAddr, err)
		http.Error(rw, "", 403)
		return
	}
	src := ts.NewDemuxer(req.Body)
	if _, err = src.Streams(); err != nil {
		log.Printf("[ts] error: %s from %s: %s", req.URL.Path, req.RemoteAddr, err)
		http.Error(rw, "", 500)
		return
	}
	if err := s.Channels.Publish(auth, "ts", req.RemoteAddr, src); err != nil {
		log.Printf("[ts] error: %s from %s: %s", req.URL.Path, req.RemoteAddr, err)
		http.Error(rw, "", 500)
		return
	}
}
