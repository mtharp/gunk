package web

import (
	"net/http"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
)

func realIPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		hlog.FromRequest(r).UpdateContext(func(c zerolog.Context) zerolog.Context {
			c = c.Str("ip", stripPort(r.RemoteAddr))
			if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
				c = c.Str("xff", xff)
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
