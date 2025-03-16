package server

import (
	"net/http"
	"net/url"

	"eaglesong.dev/gunk/internal/model"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
)

type authRequest struct {
	User     string `json:"user"`
	Password string `json:"password"`
	IP       string `json:"ip"`
	Action   string `json:"action"`
	Path     string `json:"path"`
	Protocol string `json:"protocol"`
	ID       string `json:"id"`
	Query    string `json:"query"`
}

func (s *Server) callbackAuth(rw http.ResponseWriter, req *http.Request) {
	var ar authRequest
	if !readJSON(rw, req, &ar) {
		return
	}
	hlog.FromRequest(req).UpdateContext(func(c zerolog.Context) zerolog.Context {
		c = c.Str("auth_ip", ar.IP)
		c = c.Str("auth_path", ar.Path)
		c = c.Str("auth_action", ar.Action)
		return c
	})
	switch ar.Action {
	case "publish":
		vars, _ := url.ParseQuery(ar.Query)
		password := ar.Password
		if password == "" {
			password = vars.Get("jwt")
		}
		if password == "" {
			password = vars.Get("key")
		}
		if password == "" {
			http.Error(rw, "authorization required", http.StatusUnauthorized)
			return
		}
		def, err := s.Model.VerifyPassword(req.Context(), ar.Path, password)
		if err == model.ErrUserNotFound {
			http.Error(rw, "authorization required", http.StatusUnauthorized)
		} else if err != nil {
			hlog.FromRequest(req).Err(err).Msg("error verifying password")
			http.Error(rw, "", http.StatusInternalServerError)
		} else {
			hlog.FromRequest(req).Info().Str("user_id", def.UserID).Msg("authenticated successfully")
			rw.WriteHeader(http.StatusNoContent)
		}
		return
	}
	hlog.FromRequest(req).Error().Msg("unsupported action")
	http.Error(rw, "authorization required", http.StatusUnauthorized)
}
