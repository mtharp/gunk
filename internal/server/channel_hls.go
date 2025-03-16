package server

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"
)

func (s *Server) SetHLS(u *url.URL) {
	s.HLSProxy = &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.Out.URL.Path = pr.In.URL.Path[4:]
			pr.Out.Header.Del("X-Real-Ip")
			pr.SetURL(u)
			pr.SetXForwarded()
		},
	}
}

func (s *Server) viewHLS(rw http.ResponseWriter, req *http.Request) {
	chName := req.PathValue("channel")
	clientIP := stripPort(req.RemoteAddr)
	s.hlsViews.Viewed(chName, clientIP)
	s.HLSProxy.ServeHTTP(rw, req)
}

const webViewTimeout = 16 * time.Second

type viewCounter struct {
	mu    sync.Mutex
	count map[string]map[string]time.Time
}

func (v *viewCounter) Viewed(chName, clientIP string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.count == nil {
		v.count = make(map[string]map[string]time.Time)
	}
	m := v.count[chName]
	if m == nil {
		m = make(map[string]time.Time)
		v.count[chName] = m
	}
	m[clientIP] = time.Now()
}

func (v *viewCounter) Count(chName string) int {
	var ret int
	v.mu.Lock()
	defer v.mu.Unlock()
	m := v.count[chName]
	for clientIP, t := range m {
		if time.Since(t) > webViewTimeout {
			delete(m, clientIP)
		} else {
			ret += 1
		}
	}
	return ret
}
