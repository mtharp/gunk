package playrtc

import (
	"fmt"
	"log"

	"eaglesong.dev/gunk/sinks/rtsp"
	"github.com/pion/webrtc/v2"
)

type OfferToReceive struct {
	PlayRequest
	Offer webrtc.SessionDescription
}

func (o OfferToReceive) Answer() (*Sender, error) {
	// build tracks
	streams, err := o.Source().Streams()
	if err != nil {
		return nil, err
	}
	var m webrtc.MediaEngine
	if err := m.PopulateFromSDP(o.Offer); err != nil {
		return nil, fmt.Errorf("populate from SDP: %w", err)
	}
	s, err := o.PlayRequest.newSender(m, len(streams), true)
	if err != nil {
		return nil, err
	}
	direction := chooseDirection(o.Offer)
	if err := s.setupTracks(streams, direction); err != nil {
		s.Close()
		return nil, err
	}
	// build answer
	if err := s.pc.SetRemoteDescription(o.Offer); err != nil {
		s.Close()
		return nil, err
	}
	answer, err := s.pc.CreateAnswer(nil)
	if err != nil {
		s.Close()
		return nil, err
	}
	if err := s.pc.SetLocalDescription(answer); err != nil {
		s.Close()
		return nil, err
	}
	s.sdp = answer
	// serve in background
	go s.serve()
	return s, nil
}

func (s *Sender) SDP() webrtc.SessionDescription {
	return s.sdp
}

func (s *Sender) Candidate(candidate webrtc.ICECandidateInit) {
	log.Println("got candidate:", candidate.Candidate)
	s.pc.AddICECandidate(candidate)
}

func (p PlayRequest) OfferToSend() (*Sender, error) {
	// build tracks
	streams, err := p.Source().Streams()
	if err != nil {
		return nil, err
	}
	var m webrtc.MediaEngine
	m.RegisterCodec(rtsp.OpusCodec)
	m.RegisterCodec(rtsp.H264Codec)
	s, err := p.newSender(m, len(streams), true)
	if err != nil {
		return nil, err
	}
	if err := s.setupTracks(streams, webrtc.RTPTransceiverDirectionSendonly); err != nil {
		s.Close()
		return nil, err
	}
	offer, err := s.pc.CreateOffer(nil)
	if err != nil {
		s.Close()
		return nil, err
	}
	if err := s.pc.SetLocalDescription(offer); err != nil {
		s.Close()
		return nil, err
	}
	s.sdp = offer
	return s, nil
}

func (s *Sender) SetAnswer(answer webrtc.SessionDescription) error {
	if err := s.pc.SetRemoteDescription(answer); err != nil {
		return err
	}
	log.Println("accepted answer")
	// serve in background
	go s.serve()
	return nil
}
