package web

import (
	"net/http"
	"strconv"
	"strings"

	"eaglesong.dev/gunk/model"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/hlog"
)

func (s *Server) bearerAuth(rw http.ResponseWriter, req *http.Request) (model.ChannelAuth, bool) {
	chname := mux.Vars(req)["channel"]
	authz := strings.Fields(req.Header.Get("Authorization"))
	if len(authz) != 2 || !strings.EqualFold(authz[0], "bearer") {
		rw.Header().Set("Www-Authenticate", "Bearer realm=\"channel key\"")
		http.Error(rw, "invalid authorization header", http.StatusUnauthorized)
		return model.ChannelAuth{}, false
	}
	auth, err := model.VerifyPassword(req.Context(), chname, authz[1])
	if err != nil {
		hlog.FromRequest(req).Err(err).Str("channel", chname).Msg("Bearer authentication failed")
		http.Error(rw, "invalid authorization header", http.StatusUnauthorized)
		return model.ChannelAuth{}, false
	}
	return auth, true
}

func (s *Server) viewPublishRTC(rw http.ResponseWriter, req *http.Request) {
	auth, ok := s.bearerAuth(rw, req)
	if !ok {
		return
	}
	if !strings.HasPrefix(req.Header.Get("Content-Type"), "application/sdp") {
		http.Error(rw, "expected application/sdp", http.StatusUnsupportedMediaType)
		return
	}
	offer := readRequest(rw, req)
	if offer == nil {
		return
	}
	answer, sessionID, err := s.Channels.PublishRTC(auth, offer)
	if err != nil {
		hlog.FromRequest(req).Err(err).Str("channel", auth.Name).Msg("RTC setup failed")
		http.Error(rw, "", http.StatusInternalServerError)
		return
	}
	u, err := s.router.Get("rtc_id").URL("channel", auth.Name, "id", sessionID)
	if err != nil {
		hlog.FromRequest(req).Err(err).Str("channel", auth.Name).Msg("failed to build resource path")
		http.Error(rw, "", http.StatusInternalServerError)
		return
	}
	rw.Header().Set("Content-Type", "application/sdp")
	rw.Header().Set("Content-Length", strconv.FormatInt(int64(len(answer)), 10))
	rw.Header().Set("Location", u.String())
	// rw.Header().Set("Etag", `"foobar"`)
	rw.WriteHeader(http.StatusCreated)
	n, err := rw.Write(answer)
	hlog.FromRequest(req).Info().AnErr("werr", err).Int("wbytes", n).Send()
}

func (s *Server) viewDeleteRTC(rw http.ResponseWriter, req *http.Request) {
	auth, ok := s.bearerAuth(rw, req)
	if !ok {
		return
	}
	sessionID := mux.Vars(req)["id"]
	if err := s.Channels.StopRTC(auth, sessionID); err != nil {
		hlog.FromRequest(req).Err(err).Msg("error stopping RTC ingest")
	}
	rw.WriteHeader(http.StatusNoContent)
}
