package main

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	"eaglesong.dev/gunk/internal/discord"
	"eaglesong.dev/gunk/internal/model"
	"eaglesong.dev/gunk/internal/mtx"
	"eaglesong.dev/gunk/internal/server"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sys/unix"

	_ "net/http/pprof"
)

func main() {
	_ = godotenv.Load(".env")
	_ = godotenv.Load(".env.local")
	viper.SetDefault("listen_web", ":8009")
	viper.SetDefault("listen_callbacks", ":18009")
	viper.SetDefault("hls_api", "http://127.0.0.1:8888")
	viper.SetDefault("mtx_api", "http://127.0.0.1:9997")
	viper.SetDefault("whip_api", "http://127.0.0.1:8889")
	viper.AutomaticEnv()

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
	if v := viper.GetString("log_level"); v != "" {
		level, err := zerolog.ParseLevel(v)
		if err != nil {
			log.Fatal().Err(err).Msg("invalid LOG_LEVEL")
		}
		zerolog.SetGlobalLevel(level)
	}

	// configure base URLs
	baseURL := strings.TrimSuffix(viper.GetString("base_url"), "/")
	u, err := url.Parse(baseURL)
	if err != nil || u.Scheme == "" {
		log.Fatal().Msg("invalid BASE_URL")
	}
	hlsBaseURL := strings.TrimSuffix(viper.GetString("hls_url"), "/")
	if hlsBaseURL == "" {
		hlsBaseURL = baseURL
	}
	advertiseRTSP := viper.GetString("rtsp_url")
	if advertiseRTSP == "" {
		advertiseRTSP = "rtsp://" + u.Hostname() + ":8554"
	}
	advertiseRTMP := viper.GetString("rtmp_url")
	if advertiseRTMP == "" {
		advertiseRTMP = "rtmp://" + u.Hostname()
	}

	db, err := model.Connect()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect database")
	}

	s := &server.Server{
		BaseURL:       baseURL,
		AdvertiseHLS:  hlsBaseURL,
		AdvertiseRTMP: advertiseRTMP,
		AdvertiseRTSP: advertiseRTSP,
		Model:         db,
		MtxAPI:        mtx.API{BaseURL: viper.GetString("mtx_api")},
	}
	// configure API proxies
	if u, err := url.Parse(viper.GetString("hls_api")); err != nil {
		log.Fatal().Err(err).Msg("error parsing HLS_API")
	} else {
		s.SetHLS(u)
	}
	if u, err := url.Parse(viper.GetString("whip_api")); err != nil {
		log.Fatal().Err(err).Msg("error parsing WHIP_API")
	} else {
		s.SetWHIP(u)
	}
	// configure oauth
	if clientID := viper.GetString("client_id"); clientID != "" {
		s.SetOauth(clientID, viper.GetString("client_secret"))
	} else {
		log.Fatal().Msg("CLIENT_ID must be set")
	}
	if k := viper.GetString("cookie_secret"); k == "" {
		log.Fatal().Msg("COOKIE_SECRET must be set")
	} else {
		s.SetSecret(k)
	}
	// configure webhook
	if v := viper.GetString("webhook"); v != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		webhook, err := discord.NewWebhook(ctx, v)
		cancel()
		if err != nil {
			log.Fatal().Err(err).Msgf("failed to configure webhook")
		}
		s.Webhook = webhook
	}

	ctx, _ := signal.NotifyContext(context.Background(), unix.SIGTERM, unix.SIGINT)
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		srv := &http.Server{
			Addr:         viper.GetString("listen_web"),
			Handler:      s.WebHandler(),
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 60 * time.Second,
		}
		go func() {
			<-ctx.Done()
			log.Info().Msg("closing HTTP listener")
			eg.Go(func() error {
				ctx2, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				return srv.Shutdown(ctx2)
			})
		}()
		return srv.ListenAndServe()
	})
	eg.Go(func() error {
		srv := &http.Server{
			Addr:         viper.GetString("listen_callbacks"),
			Handler:      s.CallbackHandler(),
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		}
		go func() {
			<-ctx.Done()
			eg.Go(func() error {
				ctx2, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				return srv.Shutdown(ctx2)
			})
		}()
		return srv.ListenAndServe()
	})
	log.Err(eg.Wait()).Msg("server stopped")
}
