package web

import (
	"net/http"

	"eaglesong.dev/gunk/ingest"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/hlog"
)

func (s *Server) viewPlayWeb(rw http.ResponseWriter, req *http.Request) {
	chname := mux.Vars(req)["channel"]
	err := s.Channels.ServeWeb(rw, req, chname)
	if err == ingest.ErrNoChannel {
		http.NotFound(rw, req)
	} else if err != nil {
		hlog.FromRequest(req).Err(err).Str("channel", chname).Msg("failed to serve HLS")
	}
}

func (s *Server) viewPlayTS(rw http.ResponseWriter, req *http.Request) {
	chname := mux.Vars(req)["channel"]
	err := s.Channels.ServeTS(rw, req, chname)
	if err == ingest.ErrNoChannel {
		http.NotFound(rw, req)
	} else if err != nil {
		hlog.FromRequest(req).Err(err).Str("channel", chname).Msg("failed to serve TS")
	}
}

func (s *Server) viewPlayMP4(rw http.ResponseWriter, req *http.Request) {
	chname := mux.Vars(req)["channel"]
	err := s.Channels.ServeMP4(rw, req, chname)
	if err == ingest.ErrNoChannel {
		http.NotFound(rw, req)
	} else if err != nil {
		hlog.FromRequest(req).Err(err).Str("channel", chname).Msg("failed to serve MP4")
	}
}
