package playrtc

import (
	"errors"
	"fmt"
	"io"
	"log"
	"sync/atomic"
	"time"

	"github.com/nareix/joy4/av"
	"github.com/pion/webrtc/v3"
)

var rtcConf = webrtc.Configuration{
	ICEServers: []webrtc.ICEServer{{
		URLs: []string{
			"stun:stun1.l.google.com:19302",
			"stun:stun2.l.google.com:19302",
		},
	}},
}

var api *webrtc.API

func init() {
	m := new(webrtc.MediaEngine)
	if err := m.RegisterDefaultCodecs(); err != nil {
		panic(err)
	}
	var se webrtc.SettingEngine
	// TODO: SetNAT1To1IPs
	se.SetNetworkTypes([]webrtc.NetworkType{webrtc.NetworkTypeUDP4})
	api = webrtc.NewAPI(webrtc.WithMediaEngine(m), webrtc.WithSettingEngine(se))
}

type PlayRequest struct {
	Remote        string
	Source        func() av.Demuxer
	AddViewer     func(int)
	SendCandidate func(webrtc.ICECandidateInit)
}

type Sender struct {
	pc     *webrtc.PeerConnection
	state  uintptr
	tracks []*senderTrack
	req    PlayRequest
	sdp    webrtc.SessionDescription
}

func (p PlayRequest) newSender(streams []av.CodecData) (*Sender, error) {
	pc, err := api.NewPeerConnection(rtcConf)
	if err != nil {
		return nil, err
	}
	s := &Sender{
		pc:     pc,
		tracks: make([]*senderTrack, len(streams)),
		req:    p,
	}
	pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("[rtc] %s connection state: %s", s.req.Remote, state)
		atomic.StoreUintptr(&s.state, uintptr(state))
	})
	pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate != nil && s.req.SendCandidate != nil {
			c := candidate.ToJSON()
			log.Println("sending candidate:", c.Candidate)
			s.req.SendCandidate(c)
		}
	})
	sconf := webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionSendonly}
	for i, stream := range streams {
		track, err := newSenderTrack(stream)
		if err != nil {
			pc.Close()
			return nil, err
		}
		if _, err := pc.AddTransceiverFromTrack(track, sconf); err != nil {
			pc.Close()
			return nil, err
		}
		s.tracks[i] = track
	}
	return s, nil
}

func (s *Sender) Close() {
	s.pc.Close()
}

func (s *Sender) serve() {
	if err := s.serveOnce(); err != nil {
		log.Printf("error: serving rtc to %s: %s", s.req.Remote, err)
	}
}

func (s *Sender) getState() webrtc.ICEConnectionState {
	return webrtc.ICEConnectionState(atomic.LoadUintptr(&s.state))
}

func (s *Sender) serveOnce() error {
	s.req.AddViewer(1)
	defer s.req.AddViewer(-1)
	defer s.pc.Close()
	q := s.req.Source()
	if q == nil {
		return errors.New("channel is gone")
	}
	const rtcTimeout = time.Minute
	deadline := time.Now().Add(rtcTimeout)
	for {
		packet, err := q.ReadPacket()
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
