package web

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"regexp"

	"eaglesong.dev/gunk/ui/src"
	"github.com/gorilla/mux"
)

func uiRoutes(r *mux.Router) {
	uiLoc := os.Getenv("UI")
	if uiLoc == "" {
		log.Fatalln("set UI to location of UI, either local path or URL")
	}
	u, err := url.Parse(uiLoc)
	if err != nil {
		log.Fatalln("error:", err)
	}
	var handler http.Handler
	if u.Scheme != "" {
		handler = httputil.NewSingleHostReverseProxy(u)
	} else {
		uiLoc = filepath.Clean(uiLoc)
		handler = staticServer{Root: uiLoc}
	}
	indexHandler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		req.URL.Path = "/"
		handler.ServeHTTP(rw, req)
	})
	for _, indexRoute := range src.IndexRoutes() {
		r.Handle(indexRoute, indexHandler)
	}
	r.NotFoundHandler = cacheImmutable(handler)

	// proxy avatars to avoid being blocked by privacy tools
	cdn, _ := url.Parse(discordCDN)
	avatars := httputil.NewSingleHostReverseProxy(cdn)
	r.PathPrefix("/avatars").HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		req.Host = cdn.Host
		avatars.ServeHTTP(rw, req)
	})
}

func noCache(h http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Cache-Control", "private, no-cache, must-revalidate")
		rw.Header().Set("Referrer-Policy", "no-referrer")
		rw.Header().Set("X-Content-Type-Options", "nosniff")
		h.ServeHTTP(rw, req)
	})
}

var immutableFiles = regexp.MustCompile(`\.[0-9a-fA-F]{8,}\.(css|js)$`)

func cacheImmutable(h http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if immutableFiles.MatchString(req.URL.Path) {
			rw.Header().Set("Cache-Control", "max-age=2592000, public, immutable")
		}
		h.ServeHTTP(rw, req)
	})
}
