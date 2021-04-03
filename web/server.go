package web

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	"eaglesong.dev/gunk/ingest"
	"eaglesong.dev/gunk/model"
	"github.com/gorilla/mux"
	"golang.org/x/oauth2"
)

type Server struct {
	Secure        bool     // set secure cookies
	BaseURL       string   // base URL
	HLSBase       *url.URL // base URL for web playback
	AdvertiseRTMP string   // base URL to advertise for RTMP ingest
	AdvertiseLive *url.URL // base URL to advertise for direct HTTP streams

	key    [32]byte
	router *mux.Router
	oauth  oauth2.Config

	webhookURL string
	checkGuild string

	Channels ingest.Manager
}

func (s *Server) Initialize() {
	s.Channels.PublishEvent = s.PublishEvent
	s.Channels.FTL.CheckUser = model.VerifyFTL
	s.Channels.FTL.Publish = s.Channels.Publish
	s.Channels.Initialize()
}

func (s *Server) Handler() http.Handler {
	r := mux.NewRouter()
	s.router = r
	r.HandleFunc("/ws", s.serveWS)
	// video
	r.HandleFunc("/live/{channel}.ts", corsOK(s.viewPlayTS)).Methods("GET", "OPTIONS").Name("live")
	r.HandleFunc("/live/{channel}.ts", s.viewPublishTS).Methods("PUT", "POST")
	r.HandleFunc("/live/{channel}.m3u8", corsOK(s.viewPlaylist)).Methods("GET", "HEAD", "OPTIONS")
	r.HandleFunc("/hd/{channel}/{filename}", corsOK(s.viewPlayWeb)).Methods("GET", "HEAD", "OPTIONS").Name("web")
	// RTC
	r.HandleFunc("/sdp/{channel}", s.viewPlaySDP).Methods("POST")
	// UI
	uiRoutes(r)
	r.HandleFunc("/channels.json", corsOK(s.viewChannelInfo)).Methods("GET", "HEAD", "OPTIONS")
	r.HandleFunc("/thumbs/{channel}/{timestamp}.jpg", corsOK(s.viewThumb)).Name("thumbs")
	// login
	r.HandleFunc("/oauth2/user", s.viewUser).Methods("GET")
	r.HandleFunc("/oauth2/initiate", s.viewOauthLogin).Methods("GET")
	r.HandleFunc("/oauth2/cb", s.viewOauthCB).Methods("GET")
	r.HandleFunc("/oauth2/logout", s.viewOauthLogout).Methods("POST")
	// model
	r.HandleFunc("/api/mychannels", s.viewDefs).Methods("GET")
	r.HandleFunc("/api/mychannels", s.viewDefsCreate).Methods("POST")
	r.HandleFunc("/api/mychannels/{name}", s.viewDefsUpdate).Methods("PUT")
	r.HandleFunc("/api/mychannels/{name}", s.viewDefsDelete).Methods("DELETE")
	r.HandleFunc("/health", s.viewHealth).Methods("GET")
	h := noCache(r)
	h = accessLog(h)
	h = realIPMiddleware(h)
	return h
}

func (s *Server) viewHealth(rw http.ResponseWriter, req *http.Request) {
	rw.Write([]byte("OK"))
}

func (s *Server) checkAuth(rw http.ResponseWriter, req *http.Request) string {
	var info discordUser
	err := s.unseal(req, loginCookie, &info)
	if err == nil {
		return info.ID
	}
	log.Printf("error: authentication failed for %s to %s", req.RemoteAddr, req.URL)
	http.Error(rw, "not authorized", 401)
	return ""
}

func parseRequest(rw http.ResponseWriter, req *http.Request, d interface{}) bool {
	blob, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Printf("error: reading %s request: %s", req.RemoteAddr, err)
		http.Error(rw, "", 500)
		return false
	}
	if err := json.Unmarshal(blob, d); err != nil {
		log.Printf("error: reading %s request: %s", req.RemoteAddr, err)
		http.Error(rw, "invalid JSON in request", 400)
		return false
	}
	return true
}

func writeJSON(rw http.ResponseWriter, d interface{}) {
	rw.Header().Set("Content-Type", "application/json")
	if d == nil {
		rw.Write([]byte("{}"))
		return
	}
	blob, _ := json.Marshal(d)
	rw.Write(blob)
}

func corsOK(f http.HandlerFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Origin") != "" && (req.Method == "GET" || req.Method == "HEAD" || req.Method == "OPTIONS") {
			rw.Header().Set("Access-Control-Allow-Origin", "*")
			if h := req.Header.Get("Access-Control-Request-Headers"); h != "" {
				rw.Header().Set("Access-Control-Allow-Headers", h)
			}
			rw.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
			rw.Header().Set("Access-Control-Max-Age", "86400")
		}
		f(rw, req)
	}
}
