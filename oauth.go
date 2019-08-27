// Copyright © Michael Tharp <gxti@partiallystapled.com>
//
// This file is distributed under the terms of the MIT License.
// See the LICENSE file at the top of this tree or http://opensource.org/licenses/MIT

package main

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	"golang.org/x/oauth2"
)

const (
	stateCookieExpires = 15 * 60
	loginCookieExpires = 30 * 24 * 60 * 60
)

type loginInfo struct {
	identity
}

func (s *gunkServer) viewUser(rw http.ResponseWriter, req *http.Request) {
	var info loginInfo
	if err := s.unseal(req, s.loginCookie, &info); err != nil {
		info = loginInfo{}
	}
	if info.Avatar != "" {
		info.Avatar = "/avatars/" + info.ID + "/" + info.Avatar + ".png"
	}
	blob, _ := json.Marshal(info)
	rw.Header().Set("Content-Type", "application/json")
	rw.Write(blob)
}

func (s *gunkServer) viewLogin(rw http.ResponseWriter, req *http.Request) {
	if s.oauth == nil {
		http.Error(rw, "oauth not configured", 400)
		return
	}
	sb := make([]byte, 9)
	if _, err := io.ReadFull(rand.Reader, sb); err != nil {
		panic(err)
	}
	state := base64.RawURLEncoding.EncodeToString(sb)
	s.setCookie(rw, s.stateCookie, state, stateCookieExpires)
	http.Redirect(rw, req, s.oauth.AuthCodeURL(state), http.StatusFound)
}

func (s *gunkServer) viewCB(rw http.ResponseWriter, req *http.Request) {
	if s.oauth == nil {
		http.Error(rw, "oauth not configured", 400)
		return
	}
	token, err := s.tokenExchange(rw, req)
	if err != nil {
		log.Printf("[oauth] error: %s: %s", req.RemoteAddr, err)
		http.Error(rw, "oauth failure", 400)
		return
	}
	var info loginInfo
	info.identity, err = s.getIdentity(req.Context(), token)
	if err != nil {
		log.Printf("[oauth] error: %s: %s", req.RemoteAddr, err)
		http.Error(rw, "error getting user info from discord", 400)
		return
	}
	if err := s.setCookie(rw, s.loginCookie, &info, loginCookieExpires); err != nil {
		log.Printf("[oauth] error: persisting login: %s", err)
		http.Error(rw, "error setting login cookie", 500)
		return
	}
	http.Redirect(rw, req, "/", http.StatusFound)
}

func (s *gunkServer) tokenExchange(rw http.ResponseWriter, req *http.Request) (*oauth2.Token, error) {
	code := req.FormValue("code")
	if code == "" {
		return nil, errors.New("missing code")
	}
	state := req.FormValue("state")
	var state2 string
	err := s.unseal(req, s.stateCookie, &state2)
	s.setCookie(rw, s.stateCookie, nil, -1)
	if err != nil {
		return nil, err
	} else if !hmac.Equal([]byte(state2), []byte(state)) {
		return nil, errors.New("state mismatch")
	}
	return s.oauth.Exchange(req.Context(), code)
}

func httpGet(ctx context.Context, cli *http.Client, uri string, body interface{}) error {
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return err
	}
	resp, err := cli.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	blob, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	} else if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %s on %s %s:\n%s", resp.Status, resp.Request.Method, resp.Request.URL, string(blob))
	}
	return json.Unmarshal(blob, body)
}

func (s *gunkServer) getIdentity(ctx context.Context, token *oauth2.Token) (ident identity, err error) {
	cli := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))
	if err = httpGet(ctx, cli, "https://discordapp.com/api/users/@me", &ident); err != nil {
		return
	}
	var guildList []guildItem
	if err = httpGet(ctx, cli, "https://discordapp.com/api/users/@me/guilds", &guildList); err != nil {
		return
	}
	var announce bool
	for _, guild := range guildList {
		if guild.ID == s.webhookGuild {
			announce = true
		}
	}
	_, err = db.Exec("INSERT INTO users (user_id, refresh_token, announce) VALUES ($1, $2, $3) ON CONFLICT (user_id) DO UPDATE SET refresh_token = EXCLUDED.refresh_token, announce = EXCLUDED.announce", ident.ID, token.RefreshToken, announce)
	return
}

func (s *gunkServer) viewLogout(rw http.ResponseWriter, req *http.Request) {
	s.setCookie(rw, s.loginCookie, nil, -1)
	rw.Header().Set("Content-Type", "application/json")
	rw.Write([]byte("{}"))
}

type identity struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Discriminator string `json:"discriminator"`
	Avatar        string `json:"avatar"`
}

type guildItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
