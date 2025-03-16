package server

import (
	"crypto/hmac"
	"errors"
	"net/http"

	"eaglesong.dev/gunk/internal"
	"eaglesong.dev/gunk/internal/discord"
	"github.com/rs/zerolog/hlog"
	"golang.org/x/oauth2"
)

const (
	stateCookieExpires = 15 * 60
	loginCookieExpires = 30 * 24 * 60 * 60
)

func (s *Server) SetOauth(clientID, clientSecret string) {
	s.oauth = oauth2.Config{
		RedirectURL:  s.BaseURL + "/oauth2/cb",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     discord.Endpoint,
		Scopes:       []string{"identify", "guilds"},
	}
}

// return metadata for the logged-in user
func (s *Server) viewUser(rw http.ResponseWriter, req *http.Request) {
	var info discord.User
	if err := s.validateCookie(req, loginCookie, &info); err != nil {
		info = discord.User{}
	}
	if info.Avatar != "" {
		info.Avatar = "/avatars/" + info.ID + "/" + info.Avatar + ".png"
	}
	writeJSON(rw, info)
}

// initiate oauth flow
func (s *Server) viewOauthLogin(rw http.ResponseWriter, req *http.Request) {
	state := internal.RandomID(9)
	s.setCookie(rw, stateCookie, state, stateCookieExpires)
	http.Redirect(rw, req, s.oauth.AuthCodeURL(state), http.StatusFound)
}

// handle oauth callback and complete login
func (s *Server) viewOauthCB(rw http.ResponseWriter, req *http.Request) {
	token, err := s.tokenExchange(rw, req)
	if err != nil {
		hlog.FromRequest(req).Err(err).Msg("oauth exchange failed")
		http.Error(rw, "oauth failure", http.StatusBadRequest)
		return
	}
	tsrc := oauth2.StaticTokenSource(token)
	user, err := discord.CurrentUser(req.Context(), tsrc)
	if err != nil {
		hlog.FromRequest(req).Err(err).Msg("failed to get user info")
		http.Error(rw, "error getting user info from discord", http.StatusBadRequest)
		return
	}
	if err := s.Model.CreateOrUpdateUser(req.Context(), user.ID, token); err != nil {
		hlog.FromRequest(req).Err(err).Str("user_id", user.ID).Msg("failed to persist user info")
		http.Error(rw, "error setting login cookie", http.StatusInternalServerError)
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
	err := s.validateCookie(req, stateCookie, &state2)
	s.setCookie(rw, stateCookie, nil, -1)
	if err != nil {
		return nil, err
	} else if !hmac.Equal([]byte(state2), []byte(state)) {
		return nil, errors.New("state mismatch")
	}
	return s.oauth.Exchange(req.Context(), code)
}

// clear login session
func (s *Server) viewOauthLogout(rw http.ResponseWriter, req *http.Request) {
	s.setCookie(rw, loginCookie, nil, -1)
	rw.Header().Set("Content-Type", "application/json")
	rw.Write([]byte("{}"))
}
