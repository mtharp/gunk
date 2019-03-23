// Copyright © Michael Tharp <gxti@partiallystapled.com>
//
// This file is distributed under the terms of the MIT License.
// See the LICENSE file at the top of this tree or http://opensource.org/licenses/MIT

package main

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

func (s *gunkServer) setSecret(secret string) {
	d := sha256.New()
	d.Write([]byte(secret))
	copy(s.key[:], d.Sum(nil))
}

func (s *gunkServer) setCookie(rw http.ResponseWriter, name string, value interface{}, maxAge int) error {
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

	http.SetCookie(rw, &http.Cookie{
		Name:     name,
		Value:    cvalue,
		MaxAge:   maxAge,
		Path:     "/",
		Secure:   s.cookieSecure,
		HttpOnly: true,
	})
	return nil
}

func (s *gunkServer) unseal(req *http.Request, name string, value interface{}) error {
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
