package playrtc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

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
	GatherDone    func()
}

type Sender struct {
	pc     *webrtc.PeerConnection
	state  chan webrtc.ICEConnectionState
	tracks []*senderTrack
	req    PlayRequest
}

func HandleSDP(rw http.ResponseWriter, req *http.Request, src func() av.Demuxer, addViewer func(int)) error {
	// parse offer
	blob, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return err
	}
	var offer webrtc.SessionDescription
	if err := json.Unmarshal(blob, &offer); err != nil {
		http.Error(rw, "invalid offer", 400)
		return nil
	}
	for _, line := range strings.Split(offer.SDP, "\n") {
		fmt.Println("<", line)
	}
	// build tracks
	streams, err := src().Streams()
	if err != nil {
		return err
	}
	gatherDone := make(chan struct{})
	pr := PlayRequest{
		Source:     src,
		AddViewer:  addViewer,
		Remote:     req.RemoteAddr,
		GatherDone: func() { close(gatherDone) },
	}
	s, err := pr.newSender(streams, offer)
	if err != nil {
		return err
	}
	// build answer
	if err := s.pc.SetRemoteDescription(offer); err != nil {
		s.Close()
		return err
	}
	answer, err := s.pc.CreateAnswer(nil)
	if err != nil {
		s.Close()
		return err
	}
	if err := s.pc.SetLocalDescription(answer); err != nil {
		s.Close()
		return err
	}
	<-gatherDone
	sdp := s.pc.LocalDescription()
	for _, line := range strings.Split(sdp.SDP, "\n") {
		fmt.Println(">", line)
	}
	blob, err = json.Marshal(sdp)
	if err != nil {
		s.Close()
		return err
	}
	rw.Write(blob)
	// serve in background
	go s.serve()
	return nil
}

func (p PlayRequest) newSender(streams []av.CodecData, offer webrtc.SessionDescription) (*Sender, error) {
	pc, err := api.NewPeerConnection(rtcConf)
	if err != nil {
		return nil, err
	}
	s := &Sender{
		pc:     pc,
		state:  make(chan webrtc.ICEConnectionState, 1),
		tracks: make([]*senderTrack, len(streams)),
		req:    p,
	}
	pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("[rtc] %s connection state: %s", s.req.Remote, state)
		s.state <- state
	})
	pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil && s.req.GatherDone != nil {
			s.req.GatherDone()
		} else if candidate != nil && s.req.SendCandidate != nil {
			c := candidate.ToJSON()
			log.Println("sending candidate:", c.Candidate)
			s.req.SendCandidate(c)
		}
	})
	direction := webrtc.RTPTransceiverDirectionSendonly
	if offer.SDP != "" {
		direction = chooseDirection(offer)
	}
	sconf := webrtc.RTPTransceiverInit{Direction: direction}
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

func (s *Sender) serveOnce() error {
	s.req.AddViewer(1)
	defer s.req.AddViewer(-1)
	defer s.pc.Close()
	q := s.req.Source()
	if q == nil {
		return errors.New("channel is gone")
	}
	for st := range s.state {
		if st == webrtc.ICEConnectionStateConnected {
			break
		} else if st > webrtc.ICEConnectionStateConnected {
			return fmt.Errorf("webrtc connection failed: state is %s", st)
		}
	}
	for {
		// check if still connected
		select {
		case st := <-s.state:
			if st != webrtc.ICEConnectionStateConnected {
				return nil
			}
		default:
		}

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
		_ = track.WritePacket(packet)
	}
	return nil
}
