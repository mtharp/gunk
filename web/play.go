package web

import (
	"log"
	"net/http"

	"eaglesong.dev/gunk/ingest"
	"github.com/gorilla/mux"
)

func (s *Server) viewPlayWeb(rw http.ResponseWriter, req *http.Request) {
	chname := mux.Vars(req)["channel"]
	err := s.Channels.ServeWeb(rw, req, chname)
	if err == ingest.ErrNoChannel {
		http.NotFound(rw, req)
	} else if err != nil {
		log.Println("error:", err)
	}
}

func (s *Server) viewPlayTS(rw http.ResponseWriter, req *http.Request) {
	chname := mux.Vars(req)["channel"]
	err := s.Channels.ServeTS(rw, req, chname)
	if err == ingest.ErrNoChannel {
		http.NotFound(rw, req)
	} else if err != nil {
		log.Println("error:", err)
	}
}
