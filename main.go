// Copyright © Michael Tharp <gxti@partiallystapled.com>
//
// This file is distributed under the terms of the MIT License.
// See the LICENSE file at the top of this tree or http://opensource.org/licenses/MIT

package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/gorilla/mux"
	"github.com/nareix/joy4/av/pubsub"
	"github.com/nareix/joy4/format/rtmp"
	"golang.org/x/oauth2"
	"golang.org/x/sync/errgroup"
)

type channel struct {
	queue *pubsub.Queue
	opusq *pubsub.Queue
	hls   *HLSPublisher
}

type gunkServer struct {
	channels map[string]*channel
	mu       sync.Mutex

	oauth       *oauth2.Config
	rtmp        *rtmp.Server
	rtmpBase    string
	opusBitrate int

	cookieSecure             bool
	stateCookie, loginCookie string
	key                      [32]byte
}

func main() {
	base := strings.TrimSuffix(os.Getenv("BASE_URL"), "/")
	u, err := url.Parse(base)
	if err != nil {
		log.Fatalln("error: in BASE_URL: %s", err)
	}
	s := &gunkServer{
		channels: make(map[string]*channel),
		rtmp:     &rtmp.Server{},
		rtmpBase: "rtmp://" + u.Hostname() + "/live",
	}
	s.oauth = &oauth2.Config{
		RedirectURL:  base + "/oauth2/cb",
		ClientID:     os.Getenv("CLIENT_ID"),
		ClientSecret: os.Getenv("CLIENT_SECRET"),
		Endpoint: oauth2.Endpoint{
			AuthURL:   "https://discordapp.com/api/oauth2/authorize",
			TokenURL:  "https://discordapp.com/api/oauth2/token",
			AuthStyle: oauth2.AuthStyleInHeader,
		},
		Scopes: []string{"identify"},
	}
	if u.Scheme == "https" {
		s.cookieSecure = true
		s.stateCookie = "__Host-ostate"
		s.loginCookie = "__Host-login"
	} else {
		s.stateCookie = "ostate"
		s.loginCookie = "login"
	}
	if base == "" || s.oauth.ClientID == "" || s.oauth.ClientSecret == "" {
		log.Fatalln("BASE_URL, CLIENT_ID and CLIENT_SECRET must be set")
	}
	if k := os.Getenv("COOKIE_SECRET"); k == "" {
		log.Fatalln("error: COOKIE_SECRET must be set")
	} else {
		s.setSecret(k)
	}
	if v := os.Getenv("RTMP_URL"); v != "" {
		s.rtmpBase = strings.TrimSuffix(v, "/") + "/live"
	}
	if v, _ := strconv.Atoi(os.Getenv("OPUS_BITRATE")); v > 0 {
		s.opusBitrate = v
	} else {
		s.opusBitrate = 128000
	}
	if err := connectDB(); err != nil {
		log.Fatalln("error: connecting to database:", err)
	}

	eg := new(errgroup.Group)

	s.rtmp.HandlePublish = s.handleRTMP
	eg.Go(s.rtmp.ListenAndServe)

	r := mux.NewRouter()
	// video
	r.HandleFunc("/live/{channel}.ts", s.handleTS).Methods("GET")
	r.HandleFunc("/hls/{channel}/{filename}", s.handleHLS).Methods("GET")
	// RTC
	r.HandleFunc("/sdp/{channel}", s.handleRTC).Methods("POST")
	// UI
	uiRoutes(r)
	r.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) { http.ServeFile(rw, req, "./index.html") }).Methods("GET")
	r.HandleFunc("/channels.json", s.handleChannels)
	r.HandleFunc("/thumbs/{channel}/{timestamp}.jpg", s.handleThumb)
	r.PathPrefix("/node_modules/").Handler(http.StripPrefix("/node_modules/", http.FileServer(http.Dir("./node_modules"))))
	// login
	r.HandleFunc("/oauth2/user", s.viewUser).Methods("GET")
	r.HandleFunc("/oauth2/initiate", s.viewLogin).Methods("GET")
	r.HandleFunc("/oauth2/cb", s.viewCB).Methods("GET")
	r.HandleFunc("/oauth2/logout", s.viewLogout).Methods("POST")
	// model
	r.HandleFunc("/api/mychannels", s.viewDefs).Methods("GET")
	r.HandleFunc("/api/mychannels", s.viewDefsCreate).Methods("POST")
	r.HandleFunc("/api/mychannels/{name}", s.viewDefsDelete).Methods("DELETE")

	eg.Go(func() error { return http.ListenAndServe(":8009", middleware(r)) })

	//	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	//		l.RLock()
	//		ch := channels[r.URL.Path]
	//		l.RUnlock()
	//
	//		if ch != nil {
	//			//w.Header().Set("Content-Type", "video/x-flv")
	//			//w.Header().Set("Transfer-Encoding", "chunked")
	//			w.Header().Set("Access-Control-Allow-Origin", "*")
	//			w.WriteHeader(200)
	//			//flusher := w.(http.Flusher)
	//			//flusher.Flush()
	//			//muxer := flv.NewMuxerWriteFlusher(writeFlusher{httpflusher: flusher, Writer: w})
	//			muxer := ts.NewMuxer(w)
	//			cursor := ch.que.Latest()
	//
	//			avutil.CopyFile(muxer, cursor)
	//		} else {
	//			http.NotFound(w, r)
	//		}
	//	})
	//
	//	go http.ListenAndServe(":8089", nil)

	if err := eg.Wait(); err != nil {
		log.Fatalln("error:", err)
	}
}

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
		handler = http.FileServer(http.Dir(uiLoc))
	}
	indexHandler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		req.URL.Path = "/"
		handler.ServeHTTP(rw, req)
	})
	r.Handle("/", indexHandler)
	r.Handle("/watch/{channel}", indexHandler)
	r.NotFoundHandler = handler

	// proxy avatars to avoid being blocked by privacy tools
	cdn, _ := url.Parse("https://cdn.discordapp.com")
	avatars := httputil.NewSingleHostReverseProxy(cdn)
	r.PathPrefix("/avatars").HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		req.Host = cdn.Host
		avatars.ServeHTTP(rw, req)
	})
}

func middleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Referrer-Policy", "no-referrer")
		rw.Header().Set("X-Content-Type-Options", "nosniff")
		rw.Header().Set("Access-Control-Allow-Origin", "*")
		h.ServeHTTP(rw, req)
	})
}
