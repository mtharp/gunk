package web

import (
	"net/http"
	"net/url"

	"eaglesong.dev/gunk/model"
	"github.com/gorilla/mux"
	"github.com/nareix/joy4/format/ts"
	"github.com/rs/zerolog/hlog"
)

func (s *Server) viewPublishTS(rw http.ResponseWriter, req *http.Request) {
	chname := mux.Vars(req)["channel"]
	u2 := &url.URL{
		Path:     chname,
		RawQuery: req.URL.RawQuery,
	}
	auth, err := model.VerifyRTMP(u2)
	if err != nil {
		hlog.FromRequest(req).Err(err).Str("channel", chname).Msg("TS authentication failed")
		http.Error(rw, "", http.StatusForbidden)
		return
	}
	src := ts.NewDemuxer(req.Body)
	if _, err = src.Streams(); err != nil {
		hlog.FromRequest(req).Err(err).Str("channel", chname).Msg("TS demux failed")
		http.Error(rw, "", http.StatusInternalServerError)
		return
	}
	l := hlog.FromRequest(req).With().Str("kind", "ts").Logger()
	ctx := l.WithContext(req.Context())
	if err := s.Channels.Publish(ctx, auth, src); err != nil {
		hlog.FromRequest(req).Err(err).Str("channel", chname).Msg("TS publish failed")
		http.Error(rw, "", http.StatusInternalServerError)
		return
	}
}
