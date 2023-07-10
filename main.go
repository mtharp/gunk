package main

import (
	"context"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"eaglesong.dev/gunk/ingest"
	"eaglesong.dev/gunk/ingest/irtmp"
	"eaglesong.dev/gunk/ingest/rist"
	"eaglesong.dev/gunk/model"
	"eaglesong.dev/gunk/web"
	"eaglesong.dev/hls"
	"github.com/joho/godotenv"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/format/rtmp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"

	_ "net/http/pprof"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	_ = godotenv.Load(".env")
	_ = godotenv.Load(".env.local")
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

	base := strings.TrimSuffix(viper.GetString("base_url"), "/")
	webBase, err := url.Parse(base)
	if err != nil {
		log.Fatal().Err(err).Msg("invalid BASE_URL")
	}
	liveHost := viper.GetString("live_hostname")
	if liveHost == "" {
		liveHost = webBase.Hostname()
	}
	s := &web.Server{
		BaseURL: base,
		Secure:  webBase.Scheme == "https",
		Channels: ingest.Manager{
			OpusBitrate: viper.GetInt("opus_bitrate"),
			RTCHost:     viper.GetString("rtc_host"),
		},
	}
	s.SetOauth(viper.GetString("client_id"), viper.GetString("client_secret"))
	if k := viper.GetString("cookie_secret"); k == "" {
		log.Fatal().Msg("COOKIE_SECRET must be set")
	} else {
		s.SetSecret(k)
	}
	if v := viper.GetString("webhook"); v != "" {
		if err := s.SetWebhook(v); err != nil {
			log.Fatal().Err(err).Msg("failed to set webhook")
		}
	}
	if v := viper.GetString("rtmp_url"); v != "" {
		s.AdvertiseRTMP = strings.TrimSuffix(v, "/") + "/live"
	} else {
		s.AdvertiseRTMP = "rtmp://" + liveHost + "/live"
	}
	if v := viper.GetString("live_url"); v != "" {
		s.AdvertiseLive, err = url.Parse(v)
		if err != nil {
			log.Fatal().Err(err).Msg("invalid LIVE_URL")
		}
	} else {
		s.AdvertiseLive = webBase
	}
	if v := viper.GetString("hls_url"); v != "" {
		s.HLSBase, err = url.Parse(v)
		if err != nil {
			log.Fatal().Err(err).Msg("invalid HLS_URL")
		}
	}
	switch viper.GetString("web_mode") {
	case "dash", "":
		s.Channels.PublishMode = hls.ModeSeparateTracks
	case "hls":
		s.Channels.PublishMode = hls.ModeSingleTrack
	case "both":
		s.Channels.PublishMode = hls.ModeSingleAndSeparate
	default:
		log.Fatal().Msg("WEB_MODE must be one of: dash, hls, both")
	}
	if v := viper.GetString("work_dir"); v != "" {
		if err := os.MkdirAll(v, 0700); err != nil {
			log.Fatal().Err(err).Msg("failed to create WORK_DIR")
		}
		s.Channels.WorkDir = v
	}
	if err := model.Connect(); err != nil {
		log.Fatal().Err(err).Msg("failed to connect database")
	}
	if err := s.Initialize(); err != nil {
		log.Fatal().Err(err).Msg("failed to start server")
	}

	if v := viper.GetString("metrics"); v != "" {
		lis, err := net.Listen("tcp", v)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to start metrics")
		}
		go http.Serve(lis, nil)
	}

	eg := new(errgroup.Group)
	rs := &irtmp.Server{
		Server: rtmp.Server{
			Addr: viper.GetString("listen_rtmp"),
		},
		CheckUser: func(u *url.URL) (model.ChannelAuth, error) {
			chname := path.Base(u.Path)
			key := u.Query().Get("key")
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			return model.VerifyPassword(ctx, chname, key)
		},
		Publish: s.Channels.Publish,
	}
	eg.Go(func() error { return rs.ListenAndServe() })
	if v := viper.GetString("listen_rist"); v != "" {
		ristServer := rist.New(func(ctx context.Context, name string, src av.Demuxer) error {
			ch, err := model.GetChannel(ctx, name)
			if err != nil {
				return err
			}
			return s.Channels.Publish(ctx, ch, src)
		})
		eg.Go(func() error { return ristServer.ListenAndServe(v) })
		if w := viper.GetString("advertise_rist"); w != "" {
			s.AdvertiseRIST, err = url.Parse(w)
			if err != nil {
				log.Fatal().Err(err).Msg("invalid ADVERTISE_RIST")
			}
		} else {
			_, port, err := net.SplitHostPort(v)
			if err != nil {
				log.Fatal().Err(err).Msg("unable to parse port from LISTEN_RIST")
			}
			hostport := net.JoinHostPort(liveHost, port)
			s.AdvertiseRIST = &url.URL{
				Scheme:   "rist",
				Host:     hostport,
				RawQuery: "mux-mode=1",
			}
		}
	}
	eg.Go(func() error {
		srv := &http.Server{
			Addr:              ":8009",
			Handler:           s.Handler(),
			ReadHeaderTimeout: 15 * time.Second,
		}
		return srv.ListenAndServe()
	})
	go func() {
		for range time.NewTicker(15 * time.Second).C {
			s.Channels.Cleanup()
		}
	}()
	err = eg.Wait()
	log.Err(err).Msg("server stopped")
}
