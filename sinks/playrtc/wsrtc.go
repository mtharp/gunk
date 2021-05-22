package playrtc

import (
	"log"

	"github.com/pion/webrtc/v3"
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
	s, err := o.PlayRequest.newSender(streams, o.Offer)
	if err != nil {
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
	// serve in background
	go s.serve()
	return s, nil
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
	s, err := p.newSender(streams, webrtc.SessionDescription{})
	if err != nil {
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
