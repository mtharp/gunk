package main

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"eaglesong.dev/gunk/ingest/irtmp"
	"eaglesong.dev/gunk/model"
	"eaglesong.dev/gunk/sinks/rtsp"
	"eaglesong.dev/gunk/web"
	"github.com/nareix/joy4/format/rtmp"
	"golang.org/x/sync/errgroup"
)

func main() {
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
	if v, _ := strconv.Atoi(os.Getenv("OPUS_BITRATE")); v > 0 {
		s.Channels.OpusBitrate = v
	}
	if err := model.Connect(); err != nil {
		log.Fatalln("error: connecting to database:", err)
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

	eg.Go(func() error { return http.ListenAndServe(":8009", s.Handler()) })
	if err := eg.Wait(); err != nil {
		log.Fatalln("error:", err)
	}
}
