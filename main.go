package main

import (
	"context"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"eaglesong.dev/gunk/ingest"
	"eaglesong.dev/gunk/ingest/irtmp"
	"eaglesong.dev/gunk/ingest/rist"
	"eaglesong.dev/gunk/model"
	"eaglesong.dev/gunk/web"
	"github.com/joho/godotenv"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/format/rtmp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"

	_ "net/http/pprof"
)

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		level, err := zerolog.ParseLevel(v)
		if err != nil {
			log.Fatal().Err(err).Msg("invalid LOG_LEVEL")
		}
		zerolog.SetGlobalLevel(level)
	}
	rand.Seed(time.Now().UnixNano())
	_ = godotenv.Load(".env")
	_ = godotenv.Load(".env.local")
	base := strings.TrimSuffix(os.Getenv("BASE_URL"), "/")
	u, err := url.Parse(base)
	if err != nil {
		log.Fatal().Err(err).Msg("invalid BASE_URL")
	}
	s := &web.Server{
		BaseURL: base,
		Secure:  u.Scheme == "https",
		Channels: ingest.Manager{
			RTCHost: os.Getenv("RTC_HOST"),
		},
	}
	s.SetOauth(os.Getenv("CLIENT_ID"), os.Getenv("CLIENT_SECRET"))
	if k := os.Getenv("COOKIE_SECRET"); k == "" {
		log.Fatal().Msg("COOKIE_SECRET must be set")
	} else {
		s.SetSecret(k)
	}
	if v := os.Getenv("WEBHOOK"); v != "" {
		if err := s.SetWebhook(v); err != nil {
			log.Fatal().Err(err).Msg("failed to set webhook")
		}
	}
	if v := os.Getenv("RTMP_URL"); v != "" {
		s.AdvertiseRTMP = strings.TrimSuffix(v, "/") + "/live"
	} else {
		s.AdvertiseRTMP = "rtmp://" + u.Hostname() + "/live"
	}
	if v := os.Getenv("LIVE_URL"); v != "" {
		s.AdvertiseLive, err = url.Parse(v)
		if err != nil {
			log.Fatal().Err(err).Msg("invalid LIVE_URL")
		}
	} else {
		s.AdvertiseLive = u
	}
	if v := os.Getenv("HLS_URL"); v != "" {
		s.HLSBase, err = url.Parse(v)
		if err != nil {
			log.Fatal().Err(err).Msg("invalid HLS_URL")
		}
	}
	if v := os.Getenv("WORK_DIR"); v != "" {
		if err := os.MkdirAll(v, 0700); err != nil {
			log.Fatal().Err(err).Msg("failed to create WORK_DIR")
		}
		s.Channels.WorkDir = v
	}
	if v, _ := strconv.Atoi(os.Getenv("OPUS_BITRATE")); v > 0 {
		s.Channels.OpusBitrate = v
	}
	s.Channels.UseDASH = true
	if err := model.Connect(); err != nil {
		log.Fatal().Err(err).Msg("failed to connect database")
	}
	if err := s.Initialize(); err != nil {
		log.Fatal().Err(err).Msg("failed to start server")
	}

	if v := os.Getenv("METRICS"); v != "" {
		lis, err := net.Listen("tcp", v)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to start metrics")
		}
		go http.Serve(lis, nil)
	}

	eg := new(errgroup.Group)
	rs := &irtmp.Server{
		Server: rtmp.Server{
			Addr: os.Getenv("LISTEN_RTMP"),
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
	if v := os.Getenv("LISTEN_RIST"); v != "" {
		ristServer := rist.New(func(ctx context.Context, name string, src av.Demuxer) error {
			ch, err := model.GetChannel(ctx, name)
			if err != nil {
				return err
			}
			return s.Channels.Publish(ctx, ch, src)
		})
		eg.Go(func() error { return ristServer.ListenAndServe(v) })
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
