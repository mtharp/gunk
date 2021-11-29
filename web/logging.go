package web

import (
	"net/http"
	"time"

	"github.com/rs/zerolog/hlog"
)

func accessLog(r *http.Request, status, size int, duration time.Duration) {
	if r.URL.Path == "/health" {
		return
	}
	ev := hlog.FromRequest(r).Info().
		Timestamp().
		Str("method", r.Method).
		Stringer("url", r.URL).
		Int("status", status).
		Int("length", size).
		Dur("dur", duration)
	if ua := r.Header.Get("User-Agent"); ua != "" {
		ev = ev.Str("ua", ua)
	}
	ev.Send()
}
