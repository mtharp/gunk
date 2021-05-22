package playrtc

import (
	"bytes"
	"fmt"
	"log"
	"sync"
	"time"

	"eaglesong.dev/gunk/h264util"
	"eaglesong.dev/gunk/internal"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/codec/h264parser"
	"github.com/nareix/joy4/codec/opusparser"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v3"
)

type senderTrack struct {
	*webrtc.TrackLocalStaticRTP
	cd        av.CodecData
	payloader rtp.Payloader
	mu        sync.Mutex

	packetizer rtp.Packetizer
	h264       *h264parser.CodecData

	ts   uint64
	rate uint64
	got  bool
	buf  bytes.Buffer
}

func newSenderTrack(cd av.CodecData) (*senderTrack, error) {
	// map joy4 codec to rtp codec and payloader
	s := &senderTrack{cd: cd}
	var capability webrtc.RTPCodecCapability
	switch cd := cd.(type) {
	case h264parser.CodecData:
		s.payloader = &codecs.H264Payloader{}
		s.h264 = &cd
		capability = webrtc.RTPCodecCapability{
			MimeType:    webrtc.MimeTypeH264,
			ClockRate:   90000,
			SDPFmtpLine: "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42e01f",
		}
	case *opusparser.CodecData:
		s.payloader = &codecs.OpusPayloader{}
		capability = webrtc.RTPCodecCapability{
			MimeType:  webrtc.MimeTypeOpus,
			ClockRate: 48000,
			Channels:  2,
		}
	default:
		return nil, fmt.Errorf("unsupported codec %T", cd)
	}
	// create underlying rtp track
	var err error
	s.TrackLocalStaticRTP, err = webrtc.NewTrackLocalStaticRTP(capability, cd.Type().String(), "gunk")
	if err != nil {
		return nil, err
	}
	return s, err
}

func (s *senderTrack) Bind(t webrtc.TrackLocalContext) (webrtc.RTPCodecParameters, error) {
	codec, err := s.TrackLocalStaticRTP.Bind(t)
	if err != nil {
		return codec, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	log.Println("bound", codec.PayloadType, codec.MimeType, codec.SDPFmtpLine)
	if s.packetizer != nil {
		return codec, nil
	}
	// finish creating framer
	s.packetizer = rtp.NewPacketizer(
		1200,
		0, 0,
		s.payloader,
		rtp.NewRandomSequencer(),
		codec.ClockRate,
	)
	s.rate = uint64(codec.ClockRate)
	return codec, nil
}

func (s *senderTrack) WritePacket(pkt av.Packet) error {
	s.mu.Lock()
	p := s.packetizer
	rate := s.rate
	s.mu.Unlock()
	if p == nil {
		return nil
	}
	// convert timestamp back to clock rate
	samples := s.delta(pkt.Time, rate)
	data := pkt.Data
	if s.h264 != nil {
		// convert NALUs to Annex B
		s.buf.Reset()
		h264util.WriteAnnexBPacket(&s.buf, pkt, *s.h264)
		data = s.buf.Bytes()
	}
	// packetize and send
	for _, pkt := range p.Packetize(data, samples) {
		if err := s.TrackLocalStaticRTP.WriteRTP(pkt); err != nil {
			return err
		}
	}
	return nil
}

// convert a packet duration to sample timescale accurately
func (s *senderTrack) delta(t time.Duration, rate uint64) uint32 {
	ts := internal.ToTS(t, rate)
	var samples uint32
	if s.got {
		samples = uint32(ts - s.ts)
	}
	s.ts = ts
	s.got = true
	return samples
}
