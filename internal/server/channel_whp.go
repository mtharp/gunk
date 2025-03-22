package server

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
)

var (
	// Original mtx URL takes the form /path/whep/session-id
	whpMtxURL = regexp.MustCompile(`^(/.+?)(/wh[ei]p)(/[A-Fa-f0-9]{8}-(?:[A-Fa-f0-9]{4}-){3}[A-Fa-f0-9]{12})?$`)
	// Our proxied URL takes the form /whep/path/sesion-id
	whpOurURL = regexp.MustCompile(`^(/wh[ei]p)(/.+?)(/[A-Fa-f0-9]{8}-(?:[A-Fa-f0-9]{4}-){3}[A-Fa-f0-9]{12})?$`)
)

func (s *Server) SetWHIP(u *url.URL) {
	s.WHIPProxy = &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.Out.Header.Del("X-Real-Ip")
			pr.SetURL(u)
			pr.SetXForwarded()
		},
		ModifyResponse: func(resp *http.Response) error {
			// change mtx's path format to our proxied path format
			loc := resp.Header.Get("Location")
			if loc == "" {
				return nil
			}
			parts := whpMtxURL.FindStringSubmatch(loc)
			if parts == nil {
				return nil
			}
			path, mode, sessionID := parts[1], parts[2], parts[3]
			resp.Header.Set("Location", mode+path+sessionID)
			fmt.Println("loc:", loc, "->", resp.Header.Get("Location"))
			return nil
		},
	}
}

func (s *Server) viewWhp(rw http.ResponseWriter, req *http.Request) {
	// change the proxied path to mtx's path format before proxying
	parts := whpOurURL.FindStringSubmatch(req.URL.Path)
	if parts == nil {
		http.NotFound(rw, req)
		return
	}
	mode, path, sessionID := parts[1], parts[2], parts[3]
	fmt.Println("url:", req.URL.Path, "->", path+mode+sessionID)
	req.URL.Path = path + mode + sessionID
	s.WHIPProxy.ServeHTTP(rw, req)
}
