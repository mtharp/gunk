package playrtc

import (
	"fmt"
	"io"
	"log"
	"sync/atomic"
	"time"

	"github.com/nareix/joy4/av"
	"github.com/pion/webrtc/v3"
)

type Sender struct {
	pc     *webrtc.PeerConnection
	state  uintptr
	src    av.Demuxer
	tracks []*senderTrack
	sdp    webrtc.SessionDescription

	remoteIP      string
	addViewer     ViewerFunc
	sendCandidate CandidateSender
}

func (s *Sender) start(streams []av.CodecData) error {
	// setup callbacks
	s.pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("[rtc] %s connection state: %s", s.remoteIP, state)
		atomic.StoreUintptr(&s.state, uintptr(state))
	})
	s.pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate != nil {
			c := candidate.ToJSON()
			// log.Println("sending candidate:", c.Candidate)
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
	// for _, l := range strings.Split(offer.SDP, "\n") {
	// 	log.Println("<", l)
	// }
	s.sdp = offer
	return nil
}

func (s *Sender) Close() {
	s.pc.Close()
}

func (s *Sender) serve() {
	if err := s.serveOnce(); err != nil {
		log.Printf("error: serving rtc to %s: %s", s.remoteIP, err)
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
