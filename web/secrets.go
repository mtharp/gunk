package web

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"golang.org/x/crypto/nacl/secretbox"
)

const (
	stateCookie = "ostate"
	loginCookie = "login"
)

func (s *Server) SetSecret(secret string) {
	d := sha256.New()
	d.Write([]byte(secret))
	copy(s.key[:], d.Sum(nil))
}

func (s *Server) setCookie(rw http.ResponseWriter, name string, value interface{}, maxAge int) error {
	var cvalue string
	if value != nil {
		blob, err := json.Marshal(value)
		if err != nil {
			return err
		}
		sealed := make([]byte, 24, 24+secretbox.Overhead+len(blob))
		var nonce [24]byte
		if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
			panic(err)
		}
		copy(sealed, nonce[:])
		sealed = secretbox.Seal(sealed, blob, &nonce, &s.key)
		cvalue = base64.RawURLEncoding.EncodeToString(sealed)
	}

	cookie := &http.Cookie{
		Name:     name,
		Value:    cvalue,
		MaxAge:   maxAge,
		Path:     "/",
		Secure:   s.Secure,
		HttpOnly: true,
	}
	if s.Secure {
		cookie.Name = "__Host-" + cookie.Name
	}
	http.SetCookie(rw, cookie)
	return nil
}

func (s *Server) unseal(req *http.Request, name string, value interface{}) error {
	if s.Secure {
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
	var nonce [24]byte
	copy(nonce[:], sealed)
	blob, ok := secretbox.Open(nil, sealed[24:], &nonce, &s.key)
	if !ok {
		return errors.New("bad envelope")
	}
	return json.Unmarshal(blob, value)
}
