package web

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

func accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ua := r.Header.Get("User-Agent")
		if ua == "" {
			ua = "-"
		}
		lw := &loggingWriter{ResponseWriter: rw}
		next.ServeHTTP(lw, r)
		if r.URL.Path == "/health" {
			return
		}
		log.Printf("%s \"%s %s %s\" %d %d %q %s", r.RemoteAddr, r.Method, r.URL, r.Proto, lw.status, lw.length, ua, time.Since(start))
	})
}

type loggingWriter struct {
	http.ResponseWriter
	status int
	length int64
}

func (lw *loggingWriter) WriteHeader(status int) {
	lw.ResponseWriter.WriteHeader(status)
	lw.status = status
}

func (lw *loggingWriter) Write(d []byte) (n int, err error) {
	if lw.status == 0 {
		lw.status = 200
	}
	n, err = lw.ResponseWriter.Write(d)
	if n > 0 {
		lw.length += int64(n)
	}
	return
}

func (lw *loggingWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := lw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("can't hijack %T", lw.ResponseWriter)
	}
	return h.Hijack()
}
