package web

import (
	"errors"

	"eaglesong.dev/gunk/sinks/playrtc"
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
	var p playrtc.PlayRequest
	p.Remote = remoteIP
	p.SendCandidate = func(cand webrtc.ICECandidateInit) {
		n.send <- wsMsg{
			Type:      "candidate",
			Candidate: &cand,
		}
	}
	s, err := n.server.Channels.OfferSDP(p, name)
	if err != nil {
		return err
	}
	offer := s.SDP()
	n.send <- wsMsg{
		Type: "offer",
		SDP:  &offer,
	}
	n.rtc = s
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
