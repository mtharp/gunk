package playrtc

import (
	"github.com/nareix/joy4/av"
	"github.com/pion/webrtc/v3"
)

type ViewerFunc func(int)
type CandidateSender func(webrtc.ICECandidateInit)

func (e *Engine) OfferToSend(src av.Demuxer, addViewer ViewerFunc, remoteIP string, sendCandidate CandidateSender) (*Sender, error) {
	// build tracks
	streams, err := src.Streams()
	if err != nil {
		return nil, err
	}
	pc, err := e.newConnection()
	if err != nil {
		return nil, err
	}
	s := &Sender{
		pc:            pc,
		src:           src,
		tracks:        make([]*senderTrack, len(streams)),
		remoteIP:      remoteIP,
		addViewer:     addViewer,
		sendCandidate: sendCandidate,
	}
	if err := s.start(streams); err != nil {
		pc.Close()
		return nil, err
	}
	return s, nil
}

func (s *Sender) SDP() webrtc.SessionDescription {
	return s.sdp
}

func (s *Sender) Candidate(candidate webrtc.ICECandidateInit) {
	// log.Println("got candidate:", candidate.Candidate)
	s.pc.AddICECandidate(candidate)
}

func (s *Sender) SetAnswer(answer webrtc.SessionDescription) error {
	if err := s.pc.SetRemoteDescription(answer); err != nil {
		return err
	}
	// for _, l := range strings.Split(answer.SDP, "\n") {
	// 	log.Println(">", l)
	// }
	// serve in background
	go s.serve()
	return nil
}
