package web

import (
	"crypto/hmac"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"net/http"

	"github.com/rs/zerolog/hlog"
	"golang.org/x/oauth2"
)

const (
	stateCookieExpires = 15 * 60
	loginCookieExpires = 30 * 24 * 60 * 60
)

var discordEndpoint = oauth2.Endpoint{
	AuthURL:   discordBase + "/oauth2/authorize",
	TokenURL:  discordBase + "/oauth2/token",
	AuthStyle: oauth2.AuthStyleInHeader,
}

func (s *Server) SetOauth(clientID, clientSecret string) {
	s.oauth = oauth2.Config{
		RedirectURL:  s.BaseURL + "/oauth2/cb",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     discordEndpoint,
		Scopes:       []string{"identify", "guilds"},
	}
}

func (s *Server) viewUser(rw http.ResponseWriter, req *http.Request) {
	var info discordUser
	if err := s.unseal(req, loginCookie, &info); err != nil {
		info = discordUser{}
	}
	if info.Avatar != "" {
		info.Avatar = "/avatars/" + info.ID + "/" + info.Avatar + ".png"
	}
	writeJSON(rw, info)
}

func (s *Server) viewOauthLogin(rw http.ResponseWriter, req *http.Request) {
	if s.oauth.ClientID == "" {
		http.Error(rw, "oauth not configured", http.StatusBadRequest)
		return
	}
	sb := make([]byte, 9)
	if _, err := io.ReadFull(rand.Reader, sb); err != nil {
		panic(err)
	}
	state := base64.RawURLEncoding.EncodeToString(sb)
	s.setCookie(rw, stateCookie, state, stateCookieExpires)
	http.Redirect(rw, req, s.oauth.AuthCodeURL(state), http.StatusFound)
}

func (s *Server) viewOauthCB(rw http.ResponseWriter, req *http.Request) {
	if s.oauth.ClientID == "" {
		http.Error(rw, "oauth not configured", http.StatusBadRequest)
		return
	}
	token, err := s.tokenExchange(rw, req)
	if err != nil {
		hlog.FromRequest(req).Err(err).Msg("oauth exchange failed")
		http.Error(rw, "oauth failure", http.StatusBadRequest)
		return
	}
	user, err := s.lookupUser(req.Context(), token)
	if err != nil {
		hlog.FromRequest(req).Err(err).Msg("failed to get user info")
		http.Error(rw, "error getting user info from discord", http.StatusBadRequest)
		return
	}
	if err := s.setCookie(rw, loginCookie, user, loginCookieExpires); err != nil {
		hlog.FromRequest(req).Err(err).Str("user_id", user.ID).Msg("failed to persist user info")
		http.Error(rw, "error setting login cookie", http.StatusInternalServerError)
		return
	}
	http.Redirect(rw, req, "/", http.StatusFound)
}

func (s *Server) tokenExchange(rw http.ResponseWriter, req *http.Request) (*oauth2.Token, error) {
	code := req.FormValue("code")
	if code == "" {
		return nil, errors.New("missing code")
	}
	state := req.FormValue("state")
	var state2 string
	err := s.unseal(req, stateCookie, &state2)
	s.setCookie(rw, stateCookie, nil, -1)
	if err != nil {
		return nil, err
	} else if !hmac.Equal([]byte(state2), []byte(state)) {
		return nil, errors.New("state mismatch")
	}
	return s.oauth.Exchange(req.Context(), code)
}

func (s *Server) viewOauthLogout(rw http.ResponseWriter, req *http.Request) {
	s.setCookie(rw, loginCookie, nil, -1)
	rw.Header().Set("Content-Type", "application/json")
	rw.Write([]byte("{}"))
}
