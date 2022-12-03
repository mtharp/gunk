package rist

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/format/ts"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/rs/zerolog/log"
)

type Server struct {
	Publish PublishFunc
	dmw     io.Writer
	demux   *ts.Demuxer
	mu      sync.Mutex
}

func New(pub PublishFunc) *Server {
	return &Server{
		Publish: pub,
	}
}

type PublishFunc func(ctx context.Context, name string, src av.Demuxer) error

const payloadTypeM2TS = 33

func (s *Server) ListenAndServe(addr string) error {
	lis, err := net.ListenPacket("udp", addr)
	if err != nil {
		return err
	}
	d := make([]byte, 1500)
	for {
		n, addr, err := lis.ReadFrom(d)
		if err != nil {
			log.Err(err).Msg("failed reading RIST socket")
			time.Sleep(time.Second)
			continue
		}
		var p encapsulatedPacket
		if err := p.Parse(d[:n]); err != nil {
			// TODO flood protection
			log.Warn().Stringer("src", addr).Err(err).Msg("failed to parse RIST")
			continue
		}
		switch p.Subtype {
		case subtypePacket:
			if p.Dest%2 == 0 {
				// RTP packet
				if err := s.parseRTP(&p); err != nil {
					log.Warn().Stringer("src", addr).Err(err).Msg("failed to parse RTP")
				}
			} else {
				// RTCP packet
				if err := s.parseRTCP(&p); err != nil {
					log.Warn().Stringer("src", addr).Err(err).Msg("failed to parse RTCP")
				}
			}
		case subtypeKeepAlive:
			log.Info().Msgf("keepalive %q", p.Payload)
		}
	}
}

func (s *Server) parseRTP(p *encapsulatedPacket) error {
	pkt := new(rtp.Packet)
	if err := pkt.Unmarshal(p.Payload); err != nil {
		return err
	}
	if pkt.PayloadType != payloadTypeM2TS {
		return fmt.Errorf("payload type: expected MPEG2-TS (%d), found %d", payloadTypeM2TS, pkt.PayloadType)
	}
	s.mu.Lock()
	dmw := s.dmw
	s.mu.Unlock()
	if dmw == nil {
		return nil
	}
	_, err := dmw.Write(pkt.Payload)
	return err
}

func (s *Server) parseRTCP(p *encapsulatedPacket) error {
	pkts, err := rtcp.Unmarshal(p.Payload)
	if err != nil {
		return err
	}
	for _, pkt := range pkts {
		switch p := pkt.(type) {
		case *rtcp.SourceDescription:
			var cname string
			for _, chunk := range p.Chunks {
				for _, item := range chunk.Items {
					if item.Type == rtcp.SDESCNAME {
						cname = item.Text
					}
				}
			}
			s.mu.Lock()
			if s.demux == nil {
				r, w := io.Pipe()
				s.demux = ts.NewDemuxer(r)
				s.dmw = w
				log.Info().Str("cname", cname).Msg("starting publish")
				go s.Publish(context.Background(), cname, s.demux)
			}
			s.mu.Unlock()
		case *rtcp.SenderReport:
		case *rtcp.RawPacket:
			log.Info().Msgf("unknown RTCP: %#v", p.Header())
		}
	}
	return nil
}
