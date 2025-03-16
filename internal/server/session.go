package server

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"golang.org/x/crypto/nacl/secretbox"
)

const (
	stateCookie = "ostate"
	loginCookie = "login"
)

func (s *Server) SetSecret(secret string) {
	d := sha256.New()
	d.Write([]byte(secret))
	d.Sum(s.sessionKey[:0])
}

func (s *Server) secureCookies() bool {
	return strings.HasPrefix(s.BaseURL, "https://")
}

func (s *Server) setCookie(rw http.ResponseWriter, name string, value any, maxAge int) error {
	var cvalue string
	if value != nil {
		blob, err := json.Marshal(value)
		if err != nil {
			return err
		}
		sealed := make([]byte, 24, 24+secretbox.Overhead+len(blob))
		if _, err := io.ReadFull(rand.Reader, sealed); err != nil {
			panic(err)
		}
		sealed = secretbox.Seal(sealed, blob, (*[24]byte)(sealed), &s.sessionKey)
		cvalue = base64.RawURLEncoding.EncodeToString(sealed)
	}

	cookie := &http.Cookie{
		Name:     name,
		Value:    cvalue,
		MaxAge:   maxAge,
		Path:     "/",
		Secure:   s.secureCookies(),
		HttpOnly: true,
	}
	if cookie.Secure {
		cookie.Name = "__Host-" + cookie.Name
	}
	http.SetCookie(rw, cookie)
	return nil
}

func (s *Server) validateCookie(req *http.Request, name string, value any) error {
	if s.secureCookies() {
		name = "__Host-" + name
	}
	cookie, err := req.Cookie(name)
	if err != nil {
		return err
	} else if cookie == nil || cookie.Value == "" {
		return http.ErrNoCookie
	}
	sealed, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return err
	}
	blob, ok := secretbox.Open(nil, sealed[24:], (*[24]byte)(sealed), &s.sessionKey)
	if !ok {
		return errors.New("bad envelope")
	}
	return json.Unmarshal(blob, value)
}
