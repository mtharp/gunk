package playrtc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"time"

	"eaglesong.dev/gunk/sinks/rtsp"
	"github.com/nareix/joy4/av"
	"github.com/pion/webrtc/v2"
)

const rtcIdleTime = 5 * time.Second

var rtcConf = webrtc.Configuration{
	ICEServers: []webrtc.ICEServer{{
		URLs: []string{
			"stun:stun1.l.google.com:19302",
			"stun:stun2.l.google.com:19302",
		},
	}},
}

type PlayRequest struct {
	Remote        string
	Source        func() av.Demuxer
	AddViewer     func(int)
	SendCandidate func(webrtc.ICECandidateInit)
	GatherDone    func()
}

type Sender struct {
	media  webrtc.MediaEngine
	pc     *webrtc.PeerConnection
	state  chan webrtc.ICEConnectionState
	tracks []*rtsp.TrackFramer
	req    PlayRequest
	sdp    webrtc.SessionDescription
}

func (p PlayRequest) newSender(media webrtc.MediaEngine, tracks int, trickle bool) (*Sender, error) {
	var se webrtc.SettingEngine
	se.SetTrickle(trickle)
	api := webrtc.NewAPI(webrtc.WithMediaEngine(media), webrtc.WithSettingEngine(se))
	peerConnection, err := api.NewPeerConnection(rtcConf)
	if err != nil {
		return nil, err
	}
	s := &Sender{
		media:  media,
		pc:     peerConnection,
		state:  make(chan webrtc.ICEConnectionState, 1),
		tracks: make([]*rtsp.TrackFramer, tracks),
		req:    p,
	}
	return s, nil
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
	// build tracks
	streams, err := src().Streams()
	if err != nil {
		return err
	}
	a := time.Now()
	var m webrtc.MediaEngine
	if err := m.PopulateFromSDP(offer); err != nil {
		return fmt.Errorf("populate from SDP: %w", err)
	}
	log.Println("a", time.Since(a))
	a = time.Now()
	gatherDone := make(chan struct{})
	pr := PlayRequest{
		Source:     src,
		AddViewer:  addViewer,
		Remote:     req.RemoteAddr,
		GatherDone: func() { close(gatherDone) },
	}
	s, err := pr.newSender(m, len(streams), false)
	if err != nil {
		return err
	}
	log.Println("b", time.Since(a))
	a = time.Now()
	direction := chooseDirection(offer)
	if err := s.setupTracks(streams, direction); err != nil {
		s.Close()
		return err
	}
	log.Println("c", time.Since(a))
	a = time.Now()
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
	log.Println("d", time.Since(a))
	a = time.Now()
	<-gatherDone
	log.Println("e", time.Since(a))
	a = time.Now()
	sdp := s.pc.LocalDescription()
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

func (s *Sender) setupTracks(streams []av.CodecData, direction webrtc.RTPTransceiverDirection) error {
	s.pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("[rtc] %s connection state: %s", s.req.Remote, state)
		s.state <- state
	})
	s.pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil && s.req.GatherDone != nil {
			s.req.GatherDone()
		} else if candidate != nil && s.req.SendCandidate != nil {
			c := candidate.ToJSON()
			log.Println("sending candidate:", c.Candidate)
			s.req.SendCandidate(c)
		}
	})
	sconf := webrtc.RtpTransceiverInit{Direction: direction}
	ssrc := rand.Uint32()
	for i, stream := range streams {
		codec := s.findCodec(stream)
		if codec == nil {
			return fmt.Errorf("unsupported codec %s for RTSP", stream.Type())
		}
		track, err := s.pc.NewTrack(codec.PayloadType, ssrc, randSeq(), randSeq())
		if err != nil {
			return err
		}
		if _, err := s.pc.AddTransceiverFromTrack(track, sconf); err != nil {
			return err
		}
		s.tracks[i] = &rtsp.TrackFramer{
			CodecData: stream,
			Codec:     codec,
			Track:     track,
		}
		ssrc++
	}
	return nil
}

func randSeq() string {
	letters := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]byte, 16)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
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
