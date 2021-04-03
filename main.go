package main

import (
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"eaglesong.dev/gunk/ingest/irtmp"
	"eaglesong.dev/gunk/model"
	"eaglesong.dev/gunk/sinks/rtsp"
	"eaglesong.dev/gunk/web"
	"github.com/nareix/joy4/format/rtmp"
	"golang.org/x/sync/errgroup"

	_ "net/http/pprof"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	rand.Seed(time.Now().UnixNano())
	base := strings.TrimSuffix(os.Getenv("BASE_URL"), "/")
	u, err := url.Parse(base)
	if err != nil {
		log.Fatalf("error: in BASE_URL: %s", err)
	}
	s := &web.Server{
		BaseURL: base,
		Secure:  u.Scheme == "https",
	}
	s.Initialize()
	s.SetOauth(os.Getenv("CLIENT_ID"), os.Getenv("CLIENT_SECRET"))
	if k := os.Getenv("COOKIE_SECRET"); k == "" {
		log.Fatalln("error: COOKIE_SECRET must be set")
	} else {
		s.SetSecret(k)
	}
	if v := os.Getenv("WEBHOOK"); v != "" {
		if err := s.SetWebhook(v); err != nil {
			log.Fatalln("error: setting webhook:", err)
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
			log.Fatalln("LIVE_URL:", err)
		}
	} else {
		s.AdvertiseLive = u
	}
	if v := os.Getenv("HLS_URL"); v != "" {
		s.HLSBase, err = url.Parse(v)
		if err != nil {
			log.Fatalln("HLS_URL:", err)
		}
	}
	if v := os.Getenv("WORK_DIR"); v != "" {
		if err := os.MkdirAll(v, 0700); err != nil {
			log.Fatalln("error:", err)
		}
		s.Channels.WorkDir = v
	}
	if v, _ := strconv.Atoi(os.Getenv("OPUS_BITRATE")); v > 0 {
		s.Channels.OpusBitrate = v
	}
	s.Channels.UseDASH = true
	if err := model.Connect(); err != nil {
		log.Fatalln("error: connecting to database:", err)
	}
	if v := os.Getenv("METRICS"); v != "" {
		lis, err := net.Listen("tcp", v)
		if err != nil {
			log.Fatalln("error:", err)
		}
		go http.Serve(lis, nil)
	}

	eg := new(errgroup.Group)
	rs := &irtmp.Server{
		Server: rtmp.Server{
			Addr: os.Getenv("LISTEN_RTMP"),
		},
		CheckUser: model.VerifyRTMP,
		Publish:   s.Channels.Publish,
	}
	eg.Go(func() error { return rs.ListenAndServe() })
	rtsps := &rtsp.Server{Source: s.Channels.GetRTSPSource}
	if err := rtsps.Listen(os.Getenv("LISTEN_RTSP")); err != nil {
		log.Fatalln("error:", err)
	}
	eg.Go(func() error { return rtsps.Serve() })
	if err := s.Channels.FTL.Listen(os.Getenv("LISTEN_FTL")); err != nil {
		log.Fatalln("error:", err)
	}
	eg.Go(func() error { return s.Channels.FTL.Serve() })
	eg.Go(func() error {
		srv := &http.Server{
			Addr:        ":8009",
			Handler:     s.Handler(),
			ReadTimeout: 15 * time.Second,
		}
		return srv.ListenAndServe()
	})
	go func() {
		for range time.NewTicker(15 * time.Second).C {
			s.Channels.Cleanup()
		}
	}()
	if err := eg.Wait(); err != nil {
		log.Fatalln("error:", err)
	}
}
