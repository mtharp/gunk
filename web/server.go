package web

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"

	"eaglesong.dev/gunk/ingest"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"golang.org/x/oauth2"
)

type Server struct {
	Secure        bool     // set secure cookies
	BaseURL       string   // base URL
	HLSBase       *url.URL // base URL for web playback
	AdvertiseRTMP string   // base URL to advertise for RTMP ingest
	AdvertiseLive *url.URL // base URL to advertise for direct HTTP streams
	AdvertiseRIST *url.URL

	key    [32]byte
	router *mux.Router
	oauth  oauth2.Config

	webhookURL string
	checkGuild string

	smu      sync.Mutex
	sessions map[string]*wsSession

	Channels ingest.Manager
}

func (s *Server) Initialize() error {
	s.Channels.PublishEvent = s.PublishEvent
	if err := s.Channels.Initialize(); err != nil {
		return err
	}
	s.sessions = make(map[string]*wsSession)
	go s.checkSessions()
	return nil
}

func (s *Server) Handler() http.Handler {
	r := mux.NewRouter()
	s.router = r
	r.HandleFunc("/ws", s.serveWS)
	// video
	r.HandleFunc("/live/{channel}.ts", corsOK(s.viewPlayTS)).Methods("GET", "OPTIONS").Name("live")
	r.HandleFunc("/live/{channel}.ts", s.viewPublishTS).Methods("PUT", "POST")
	r.HandleFunc("/live/{channel}.mp4", corsOK(s.viewPlayMP4)).Methods("GET", "OPTIONS")
	r.HandleFunc("/rtc/{channel}", s.viewPublishRTC).Methods("POST")
	r.HandleFunc("/rtc/{channel}/{id}", s.viewDeleteRTC).Methods("DELETE").Name("rtc_id")
	r.HandleFunc("/live/{channel}.m3u8", corsOK(s.viewPlaylist)).Methods("GET", "HEAD", "OPTIONS")
	r.HandleFunc("/hd/{channel}/{filename}", corsOK(s.viewPlayWeb)).Methods("GET", "HEAD", "OPTIONS").Name("web")
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
	h = hlog.AccessHandler(accessLog)(h)
	h = realIPMiddleware(h)
	access := zerolog.New(os.Stderr)
	h = hlog.NewHandler(access)(h)
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
	hlog.FromRequest(req).Err(err).Msg("authentication failed")
	http.Error(rw, "not authorized", http.StatusUnauthorized)
	return ""
}

func readRequest(rw http.ResponseWriter, req *http.Request) []byte {
	const maxBody = 100e3
	if req.ContentLength > maxBody {
		http.Error(rw, "", http.StatusRequestEntityTooLarge)
		return nil
	}
	blob, err := io.ReadAll(io.LimitReader(req.Body, maxBody+1))
	if err != nil {
		http.Error(rw, "io error", http.StatusBadRequest)
		return nil
	} else if len(blob) >= maxBody {
		http.Error(rw, "", http.StatusRequestEntityTooLarge)
		return nil
	}
	return blob
}

func parseRequest(rw http.ResponseWriter, req *http.Request, d interface{}) bool {
	if blob := readRequest(rw, req); blob == nil {
		return false
	} else if err := json.Unmarshal(blob, d); err != nil {
		hlog.FromRequest(req).Err(err).Msg("error parsing request")
		http.Error(rw, "invalid JSON in request", 400)
		return false
	}
	return true
}

func writeJSON(rw http.ResponseWriter, d any) {
	var blob []byte
	if d == nil {
		blob = []byte("{}")
	} else {
		blob, _ = json.Marshal(d)
	}
	rw.Header().Set("Content-Length", strconv.FormatInt(int64(len(blob)), 10))
	rw.Header().Set("Content-Type", "application/json")
	_, _ = rw.Write(blob)
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
