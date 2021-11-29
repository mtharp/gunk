package playrtc

import (
	"fmt"
	"io"
	"sync/atomic"
	"time"

	"github.com/nareix/joy4/av"
	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog"
)

type Sender struct {
	pc     *webrtc.PeerConnection
	state  uintptr
	src    av.Demuxer
	tracks []*senderTrack
	sdp    webrtc.SessionDescription

	addViewer     ViewerFunc
	sendCandidate CandidateSender
	log           zerolog.Logger
	lastIP        atomic.Value
}

func (s *Sender) start(streams []av.CodecData) error {
	// setup callbacks
	s.pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		if state == webrtc.ICEConnectionStateConnected {
			if ip := s.remoteIP(); ip != "" {
				s.lastIP.Store(ip)
			}
		}
		atomic.StoreUintptr(&s.state, uintptr(state))
		s.log.Info().Stringer("rtc_state", state).Send()
	})
	s.pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate != nil {
			c := candidate.ToJSON()
			s.log.Debug().Str("rtc_cand_sent", c.Candidate).Send()
			s.sendCandidate(c)
		}
	})
	// setup tracks
	sconf := webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionSendonly}
	for i, stream := range streams {
		track, err := newSenderTrack(stream)
		if err != nil {
			return err
		}
		if _, err := s.pc.AddTransceiverFromTrack(track, sconf); err != nil {
			return err
		}
		s.tracks[i] = track
	}
	// create initial offer
	offer, err := s.pc.CreateOffer(nil) //&webrtc.OfferOptions{ICERestart: true})
	if err != nil {
		s.Close()
		return err
	}
	if err := s.pc.SetLocalDescription(offer); err != nil {
		s.Close()
		return err
	}
	s.log.Debug().Str("rtc_offer_sent", offer.SDP).Send()
	s.sdp = offer
	return nil
}

func (s *Sender) remoteIP() string {
	for _, t := range s.pc.GetTransceivers() {
		send := t.Sender()
		if send == nil {
			continue
		}
		pair, err := send.Transport().ICETransport().GetSelectedCandidatePair()
		if err != nil {
			s.log.Warn().Err(err).Msg("failed to parse candidate pair")
			continue
		} else if pair == nil {
			continue
		}
		return pair.Remote.Address
	}
	return ""
}

func (s *Sender) Close() {
	s.pc.Close()
}

func (s *Sender) serve() {
	if err := s.serveOnce(); err != nil {
		s.log.Err(err).Msg("failed serving RTC")
	}
}

func (s *Sender) getState() webrtc.ICEConnectionState {
	return webrtc.ICEConnectionState(atomic.LoadUintptr(&s.state))
}

func (s *Sender) serveOnce() error {
	s.addViewer(1)
	defer s.addViewer(-1)
	defer s.Close()
	const rtcTimeout = time.Minute
	deadline := time.Now().Add(rtcTimeout)
	for {
		packet, err := s.src.ReadPacket()
		if err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("read error: %s", err)
		}
		track := s.tracks[int(packet.Idx)]
		if track == nil {
			continue
		}
		// check if RTC is still connected
		switch s.getState() {
		case webrtc.ICEConnectionStateConnected:
			_ = track.WritePacket(packet)
			deadline = time.Now().Add(rtcTimeout)
		case webrtc.ICEConnectionStateClosed:
			return nil
		default:
			if time.Now().After(deadline) {
				return fmt.Errorf("webrtc connection failed: %s", s.getState())
			}
		}
	}
	return nil
}
