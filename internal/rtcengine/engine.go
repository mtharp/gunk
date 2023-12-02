package rtcengine

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/pion/ice/v2"
	"github.com/pion/interceptor"
	"github.com/pion/interceptor/pkg/stats"
	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog/log"
)

type Engine struct {
	AdvertiseHost string
	ReceiveWindow time.Duration

	resolver *net.Resolver
	media    *webrtc.MediaEngine
	mux      ice.UDPMux
	conf     webrtc.Configuration
}

func New(advertiseHost string) (*Engine, error) {
	m := new(webrtc.MediaEngine)
	if err := m.RegisterDefaultCodecs(); err != nil {
		return nil, err
	}
	e := &Engine{
		media: m,
		conf: webrtc.Configuration{
			ICEServers: []webrtc.ICEServer{{
				URLs: []string{
					"stun:stun1.l.google.com:19302",
					"stun:stun2.l.google.com:19302",
				},
			}},
		},
	}
	if advertiseHost != "" {
		i := strings.IndexByte(advertiseHost, ':')
		if i > 0 {
			// bind a single port for ICE
			port, err := strconv.ParseUint(advertiseHost[i+1:], 10, 16)
			if err != nil {
				return nil, fmt.Errorf("RTC host %s has invalid port: %w", advertiseHost, err)
			}
			conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: int(port)})
			if err != nil {
				return nil, fmt.Errorf("listening on RTC host: %w", err)
			}
			e.mux = webrtc.NewICEUDPMux(nil, conn)
			advertiseHost = advertiseHost[:i]
		}
		e.AdvertiseHost = advertiseHost
		// test that the hostname exists. it will be re-resolved each time a
		// connection is started.
		resolver := &net.Resolver{PreferGo: true}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		ips, err := e.resolver.LookupHost(ctx, advertiseHost)
		if err != nil || len(ips) == 0 {
			return nil, fmt.Errorf("looking up RTC host %s: %w", advertiseHost, err)
		}
		e.resolver = resolver
	}
	return e, nil
}

func (e *Engine) Connection() (*webrtc.PeerConnection, stats.Getter, error) {
	var se webrtc.SettingEngine
	types := []webrtc.NetworkType{webrtc.NetworkTypeUDP4}
	if e.AdvertiseHost != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		ips, err := e.resolver.LookupHost(ctx, e.AdvertiseHost)
		if err == nil && len(ips) > 0 {
			se.SetNAT1To1IPs(ips, webrtc.ICECandidateTypeHost)
			var hasV4, hasV6 bool
			for _, ip := range ips {
				if strings.ContainsRune(ip, ':') {
					hasV6 = true
				} else {
					hasV4 = true
				}
			}
			types = nil
			if hasV4 {
				types = append(types, webrtc.NetworkTypeUDP4)
			}
			if hasV6 {
				types = append(types, webrtc.NetworkTypeUDP6)
			}
		} else {
			log.Err(err).Str("host", e.AdvertiseHost).Msg("failed looking up RTC host")
		}
	}
	se.SetNetworkTypes(types)
	se.SetICEMulticastDNSMode(ice.MulticastDNSModeDisabled)
	conf := e.conf
	if e.mux != nil {
		se.SetICEUDPMux(e.mux)
		// disable STUN when bound to a known port
		se.SetLite(true)
		conf.ICEServers = nil
	}
	intReg := new(interceptor.Registry)
	if err := webrtc.RegisterDefaultInterceptors(e.media, intReg); err != nil {
		return nil, nil, fmt.Errorf("configuring webrtc interceptors: %w", err)
	}
	sFact, err := stats.NewInterceptor()
	if err != nil {
		return nil, nil, fmt.Errorf("configuring stats interceptor: %w", err)
	}
	var sgetter stats.Getter
	sFact.OnNewPeerConnection(func(s string, g stats.Getter) {
		sgetter = g
	})
	intReg.Add(sFact)
	api := webrtc.NewAPI(
		webrtc.WithMediaEngine(e.media),
		webrtc.WithInterceptorRegistry(intReg),
		webrtc.WithSettingEngine(se),
	)
	pc, err := api.NewPeerConnection(conf)
	return pc, sgetter, err
}
