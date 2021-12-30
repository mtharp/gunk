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
				r.RemoteAddr = proxies[0]
			} else {
				r.RemoteAddr = clientIP
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
