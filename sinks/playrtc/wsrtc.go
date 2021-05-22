package playrtc

import (
	"log"

	"github.com/pion/webrtc/v3"
)

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
	s, err := p.newSender(streams)
	if err != nil {
		return nil, err
	}
	offer, err := s.pc.CreateOffer(nil) //&webrtc.OfferOptions{ICERestart: true})
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
