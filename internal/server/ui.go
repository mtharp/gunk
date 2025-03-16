package server

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"path/filepath"
	"strings"

	"eaglesong.dev/gunk/ui/src/router"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func uiRoutes(r *http.ServeMux) {
	uiLoc := viper.GetString("ui")
	if uiLoc == "" {
		log.Fatal().Msg("set UI to location of UI, either local path or URL")
	}
	u, err := url.Parse(uiLoc)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse UI")
	}
	var handler http.Handler
	if u.Scheme != "" {
		handler = httputil.NewSingleHostReverseProxy(u)
	} else {
		uiLoc = filepath.Clean(uiLoc)
		handler = staticServer{Root: uiLoc}
		r.Handle("GET /assets/", cacheImmutable(handler))
	}
	indexHandler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		req.URL.Path = "/"
		handler.ServeHTTP(rw, req)
	})
	for _, indexRoute := range router.IndexRoutes() {
		if strings.HasSuffix(indexRoute, "/") {
			// don't catch everything
			indexRoute += "{$}"
		}
		r.Handle("GET "+indexRoute, indexHandler)
	}
	// favicons, src and websockets for dev mode, etc.
	r.Handle("/", handler)
}
