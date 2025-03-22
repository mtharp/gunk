package server

import (
	"context"
	"net/http"
	"net/http/httputil"
	"os"
	"time"

	"eaglesong.dev/gunk/internal/discord"
	"eaglesong.dev/gunk/internal/model"
	"eaglesong.dev/gunk/internal/mtx"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"golang.org/x/oauth2"
)

type Server struct {
	Model Model
	// base URLs
	BaseURL       string
	AdvertiseHLS  string
	AdvertiseRTSP string
	AdvertiseRTMP string
	TcodePrefix   string
	// webhook
	Webhook *discord.Webhook
	// proxies
	HLSProxy  *httputil.ReverseProxy
	WHIPProxy *httputil.ReverseProxy
	// muxer
	MtxAPI mtx.API
	// oauth
	oauth      oauth2.Config
	sessionKey [32]byte
	// count HLS viewers
	hlsViews viewCounter
}

type Model interface {
	CreateOrUpdateUser(ctx context.Context, userID string, refreshToken *oauth2.Token) error
	ListRecentChannels(ctx context.Context) ([]*model.ChannelInfo, error)
	VerifyPassword(ctx context.Context, channel, password string) (model.ChannelDef, error)
	GetThumb(ctx context.Context, channelName string) ([]byte, time.Time, error)
	PutThumb(ctx context.Context, channelName string, blob []byte) (bool, time.Time, error)

	ListUserChannels(ctx context.Context, userID string) ([]*model.ChannelDef, error)
	CreateChannelDef(ctx context.Context, userID, name string) (*model.ChannelDef, error)
	UpdateChannelDef(ctx context.Context, userID, name string, announce bool) (*model.ChannelDef, error)
	DeleteChannelDef(ctx context.Context, userID, name string) error
}

func (s *Server) WebHandler() http.Handler {
	r := http.NewServeMux()
	r.HandleFunc("GET /health", viewHealth)
	// oauth
	r.HandleFunc("GET /oauth2/user", s.viewUser)
	r.HandleFunc("GET /oauth2/initiate", s.viewOauthLogin)
	r.HandleFunc("GET /oauth2/cb", s.viewOauthCB)
	r.HandleFunc("POST /oauth2/logout", s.viewOauthLogout)
	// management
	r.HandleFunc("GET /api/mychannels", s.checkAuth(s.viewMyChannels))
	r.HandleFunc("POST /api/mychannels", s.checkAuth(s.viewChannelCreate))
	r.HandleFunc("PUT /api/mychannels/{channel}", s.checkAuth(s.viewChannelUpdate))
	r.HandleFunc("DELETE /api/mychannels/{channel}", s.checkAuth(s.viewChannelDelete))
	// UI
	r.HandleFunc("GET /avatars/", discord.ProxyCDN())
	uiRoutes(r)
	// publishers
	r.HandleFunc("/whip/{path...}", s.viewWhp)
	// consumers
	r.HandleFunc("GET /channels.json", s.viewChannels)
	r.HandleFunc("GET /live/{channel}", s.viewPlaylist)
	r.HandleFunc("GET /thumbs/{channel}/{timestamp}", s.viewThumbnail)
	r.HandleFunc("/whep/{path...}", s.viewWhp)
	r.HandleFunc("/hls/{channel}/{path...}", s.viewHLS)

	var h http.Handler = r
	h = cacheNever(h)
	h = hlog.AccessHandler(accessLog)(h)
	h = realIPMiddleware(h)
	access := zerolog.New(os.Stderr)
	h = hlog.NewHandler(access)(h)
	return h
}

func (s *Server) CallbackHandler() http.Handler {
	r := http.NewServeMux()
	r.HandleFunc("GET /health", viewHealth)
	r.HandleFunc("PUT /cb/{channel}/thumb.jpeg", s.callbackThumbnail)
	r.HandleFunc("POST /cb/auth", s.callbackAuth)

	var h http.Handler = r
	h = cacheNever(h)
	h = hlog.AccessHandler(accessLog)(h)
	h = realIPMiddleware(h)
	access := zerolog.New(os.Stderr)
	h = hlog.NewHandler(access)(h)
	return h
}

func viewHealth(rw http.ResponseWriter, req *http.Request) {
	rw.Write([]byte("OK"))
}
