package web

import (
	"errors"
	"sync"

	"github.com/pion/webrtc/v3"
)

func (n *wsSession) Play(name, remoteIP string) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.rtc != nil {
		// terminate existing RTC
		n.rtc.Close()
		n.rtc = nil
	}
	// candidates can arrive before the offer is even sent, queue them up to
	// always deliver the offer first
	var mu sync.Mutex
	var sent bool
	var saved []wsMsg
	cand := func(cand webrtc.ICECandidateInit) {
		m := wsMsg{
			Type:      "candidate",
			Candidate: &cand,
		}
		mu.Lock()
		if sent {
			n.send <- m
		} else {
			saved = append(saved, m)
		}
		mu.Unlock()
	}
	s, err := n.server.Channels.OfferSDP(name, remoteIP, cand)
	if err != nil {
		return err
	}
	offer := s.SDP()
	n.send <- wsMsg{
		Type: "offer",
		SDP:  &offer,
	}
	n.rtc = s
	mu.Lock()
	sent = true
	for _, m := range saved {
		n.send <- m
	}
	saved = nil
	mu.Unlock()
	return nil
}

func (n *wsSession) Candidate(candidate *webrtc.ICECandidateInit) error {
	if candidate == nil || candidate.Candidate == "" {
		return errors.New("missing candidate")
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.rtc != nil {
		n.rtc.Candidate(*candidate)
	}
	return nil
}

func (n *wsSession) Answer(answer *webrtc.SessionDescription) error {
	if answer == nil || answer.SDP == "" {
		return errors.New("missing sdp")
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.rtc == nil {
		return errors.New("no session")
	}
	return n.rtc.SetAnswer(*answer)
}

func (n *wsSession) Stop() error {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.rtc != nil {
		n.rtc.Close()
		n.rtc = nil
	}
	return nil
}
