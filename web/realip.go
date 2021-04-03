package web

import (
	"net/http"
	"strings"
)

func realIPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			if i := strings.IndexByte(xff, ','); i > 0 {
				xff = xff[:i]
			}
			r.RemoteAddr = strings.TrimSpace(xff)
		} else {
			// remove port
			i := strings.LastIndexByte(r.RemoteAddr, ':')
			j := strings.LastIndexByte(r.RemoteAddr, ']')
			if i > j {
				r.RemoteAddr = r.RemoteAddr[:i]
			}
		}
		next.ServeHTTP(rw, r)
	})
}
