package discord

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

// Return a HTTP handler that passes through to the discord CDN
func ProxyCDN() http.HandlerFunc {
	cdn, _ := url.Parse(discordCDN)
	proxy := httputil.NewSingleHostReverseProxy(cdn)
	return func(w http.ResponseWriter, r *http.Request) {
		r.Host = cdn.Host // fix Host header
		proxy.ServeHTTP(w, r)
	}
}
