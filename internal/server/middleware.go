package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"eaglesong.dev/gunk/internal/discord"
	"eaglesong.dev/gunk/internal/model"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
)

func (s *Server) checkAuth(f func(http.ResponseWriter, *http.Request, string) error) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		var info discord.User
		if err := s.validateCookie(req, loginCookie, &info); err != nil {
			hlog.FromRequest(req).UpdateContext(func(c zerolog.Context) zerolog.Context {
				return c.AnErr("auth_error", err)
			})
			http.Error(rw, "not authorized", http.StatusUnauthorized)
			return
		}
		err := f(rw, req, info.ID)
		if err == nil {
			return
		} else if e := new(model.DuplicateChannelError); errors.As(err, e) {
			hlog.FromRequest(req).UpdateContext(func(c zerolog.Context) zerolog.Context {
				return c.AnErr("request_error", err)
			})
			http.Error(rw, e.Error(), http.StatusConflict)
		} else {
			hlog.FromRequest(req).Err(err).Msg("unhandled error")
			http.Error(rw, "", http.StatusInternalServerError)
		}
	}
}

func writeJSON(rw http.ResponseWriter, d any) {
	var blob []byte
	if d == nil {
		blob = []byte("{}")
	} else {
		blob, _ = json.Marshal(d)
	}
	rw.Header().Set("Content-Length", strconv.FormatInt(int64(len(blob)), 10))
	rw.Header().Set("Content-Type", "application/json")
	_, _ = rw.Write(blob)
}

func readRequest(rw http.ResponseWriter, req *http.Request, maxBody int64) []byte {
	if req.ContentLength > maxBody {
		http.Error(rw, "", http.StatusRequestEntityTooLarge)
		return nil
	}
	blob, err := io.ReadAll(io.LimitReader(req.Body, maxBody+1))
	if err != nil {
		http.Error(rw, "io error", http.StatusBadRequest)
		return nil
	} else if int64(len(blob)) >= maxBody {
		http.Error(rw, "", http.StatusRequestEntityTooLarge)
		return nil
	}
	return blob
}

func readJSON(rw http.ResponseWriter, req *http.Request, body any) bool {
	blob := readRequest(rw, req, 100e3)
	if len(blob) == 0 {
		return false
	}
	if err := json.Unmarshal(blob, body); err != nil {
		hlog.FromRequest(req).Err(err).Msg("failed to unmarshal request payload")
		http.Error(rw, "decode failed", http.StatusBadRequest)
		return false
	}
	return true
}

func cacheNever(h http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Cache-Control", "private, no-cache, must-revalidate")
		rw.Header().Set("Referrer-Policy", "no-referrer")
		rw.Header().Set("X-Content-Type-Options", "nosniff")
		h.ServeHTTP(rw, req)
	})
}

func cacheImmutable(h http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Cache-Control", "max-age=2592000, public, immutable")
		h.ServeHTTP(rw, req)
	})
}

func realIPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		hlog.FromRequest(r).UpdateContext(func(c zerolog.Context) zerolog.Context {
			clientIP := stripPort(r.RemoteAddr)
			c = c.Str("ip", clientIP)
			if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
				proxies := strings.Split(xff, ",")
				for i, p := range proxies {
					proxies[i] = strings.TrimSpace(p)
				}
				c = c.Strs("xff", proxies)
				// use the first IP as the "real" one for purposes of counting
				// distinct HLS viewers
				r.RemoteAddr = proxies[0] + ":0"
			}
			return c
		})
		next.ServeHTTP(rw, r)
	})
}

func stripPort(ip string) string {
	i := strings.LastIndexByte(ip, ':')
	j := strings.LastIndexByte(ip, ']')
	if i > j {
		ip = ip[:i]
	}
	if j > 0 {
		return ip[1 : len(ip)-1]
	}
	return ip
}
