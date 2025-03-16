package server

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

func (s *Server) SetWHIP(u *url.URL) {
	s.WHIPProxy = &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.Out.Header.Del("X-Real-Ip")
			pr.SetURL(u)
			pr.SetXForwarded()
		},
		ModifyResponse: func(resp *http.Response) error {
			// /foo/whip/asdf -> /whip/foo/asdf
			vv := resp.Header["Location"]
			for i, v := range vv {
				if swapped := urlSwapWhip(v); swapped != "" {
					vv[i] = swapped
				}
			}
			return nil
		},
	}
}

func (s *Server) viewWhp(rw http.ResponseWriter, req *http.Request) {
	// /whip/foo -> /foo/whip
	if swapped := urlSwapWhip(req.URL.Path); swapped == "" {
		http.NotFound(rw, req)
		return
	} else {
		req.URL.Path = swapped
	}
	s.WHIPProxy.ServeHTTP(rw, req)
}

func urlSwapWhip(url string) string {
	parts := strings.SplitN(url, "/", 4)
	if len(parts) < 3 {
		return ""
	}
	parts[1], parts[2] = parts[2], parts[1]
	return strings.Join(parts, "/")
}
