package playrtc

import (
	"context"

	"github.com/nareix/joy4/av"
	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type ViewerFunc func(int)
type CandidateSender func(webrtc.ICECandidateInit)

func (e *Engine) OfferToSend(ctx context.Context, src av.Demuxer, addViewer ViewerFunc, sendCandidate CandidateSender) (*Sender, error) {
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
		addViewer:     addViewer,
		sendCandidate: sendCandidate,
	}
	s.log = log.Ctx(ctx).Hook(zerolog.HookFunc(func(e *zerolog.Event, level zerolog.Level, message string) {
		ip, _ := s.lastIP.Load().(string)
		if ip != "" {
			e.Str("rtc_ip", ip)
		}
	}))
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
	log.Debug().Str("rtc_cand_recv", candidate.Candidate).Send()
	s.pc.AddICECandidate(candidate)
}

func (s *Sender) SetAnswer(answer webrtc.SessionDescription) error {
	if err := s.pc.SetRemoteDescription(answer); err != nil {
		return err
	}
	log.Debug().Str("rtc_answer_recv", answer.SDP).Send()
	// serve in background
	go s.serve()
	return nil
}
