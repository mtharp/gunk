// Copyright © Michael Tharp <gxti@partiallystapled.com>
//
// This file is distributed under the terms of the MIT License.
// See the LICENSE file at the top of this tree or http://opensource.org/licenses/MIT

package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sort"
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
	hls   *HLSPublisher
}

type gunkServer struct {
	channels map[string]*channel
	mu       sync.Mutex

	oauth *oauth2.Config
	rtmp  *rtmp.Server

	cookieSecure             bool
	stateCookie, loginCookie string
	key                      [32]byte
}

func main() {
	s := &gunkServer{
		channels: make(map[string]*channel),
		rtmp:     &rtmp.Server{},
	}
	if b := os.Getenv("BASE_URL"); b != "" {
		b = strings.TrimSuffix(b, "/")
		s.oauth = &oauth2.Config{
			RedirectURL:  b + "/oauth2/cb",
			ClientID:     os.Getenv("CLIENT_ID"),
			ClientSecret: os.Getenv("CLIENT_SECRET"),
			Endpoint: oauth2.Endpoint{
				AuthURL:   "https://discordapp.com/api/oauth2/authorize",
				TokenURL:  "https://discordapp.com/api/oauth2/token",
				AuthStyle: oauth2.AuthStyleInHeader,
			},
			Scopes: []string{"identify"},
		}
		if strings.HasPrefix(b, "https:") {
			s.cookieSecure = true
			s.stateCookie = "__Host-ostate"
			s.loginCookie = "__Host-login"
		} else {
			s.stateCookie = "ostate"
			s.loginCookie = "login"
		}
		if k := os.Getenv("COOKIE_SECRET"); k != "" {
			log.Fatalln("error: COOKIE_SECRET must be set")
		} else {
			s.setSecret(k)
		}
	} else {
		log.Printf("warning: oauth not configured; set BASE_URL and CLIENT_ID and CLIENT_SECRET")
	}
	eg := new(errgroup.Group)

	s.rtmp.HandlePublish = s.handleRTMP
	eg.Go(s.rtmp.ListenAndServe)

	r := mux.NewRouter()
	// HLS
	r.HandleFunc("/hls/{channel}/{filename}", s.handleHLS).Methods("GET")
	// RTC
	r.HandleFunc("/sdp/{channel}", s.handleRTC).Methods("POST")
	// UI
	uiRoutes(r)
	r.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) { http.ServeFile(rw, req, "./index.html") }).Methods("GET")
	r.HandleFunc("/channels.json", s.handleChannels)
	r.PathPrefix("/node_modules/").Handler(http.StripPrefix("/node_modules/", http.FileServer(http.Dir("./node_modules"))))
	// login
	r.HandleFunc("/oauth2/user", s.viewUser)
	r.HandleFunc("/oauth2/initiate", s.viewLogin)
	r.HandleFunc("/oauth2/cb", s.viewCB)
	r.HandleFunc("/oauth2/logout", s.viewLogout)

	eg.Go(func() error { return http.ListenAndServe(":8009", r) })

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

func (s *gunkServer) handleHLS(rw http.ResponseWriter, req *http.Request) {
	chname := mux.Vars(req)["channel"]
	s.mu.Lock()
	ch := s.channels[chname]
	s.mu.Unlock()
	if ch == nil {
		log.Printf("not found: %s", req.URL)
		http.NotFound(rw, req)
		return
	}
	ch.hls.ServeHTTP(rw, req)
}

func (s *gunkServer) handleRTC(rw http.ResponseWriter, req *http.Request) {
	chname := mux.Vars(req)["channel"]
	s.mu.Lock()
	ch := s.channels[chname]
	s.mu.Unlock()
	if ch == nil {
		log.Printf("not found: %s", req.URL)
		http.NotFound(rw, req)
		return
	}
	if err := handleSDP(rw, req, ch.queue); err != nil {
		log.Printf("error: failed to start webrtc session to %s: %s", req.RemoteAddr, err)
		http.Error(rw, "failed to start webrtc session", 500)
	}
}

func (s *gunkServer) handleChannels(rw http.ResponseWriter, req *http.Request) {
	chNames := []string{}
	s.mu.Lock()
	for chName := range s.channels {
		chNames = append(chNames, chName)
	}
	s.mu.Unlock()
	sort.Strings(chNames)
	blob, _ := json.Marshal(chNames)
	rw.Header().Set("Content-Type", "application/json")
	rw.Write(blob)
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
	r.Handle("/hls/{channel}", indexHandler)
	r.Handle("/rtc/{channel}", indexHandler)
	r.NotFoundHandler = handler
}
