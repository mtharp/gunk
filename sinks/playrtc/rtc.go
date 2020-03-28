package playrtc

import (
	"encoding/json"
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

type rtcSender struct {
	media  webrtc.MediaEngine
	pc     *webrtc.PeerConnection
	state  chan webrtc.ICEConnectionState
	tracks []*rtsp.TrackFramer
	addr   string
}

func HandleSDP(rw http.ResponseWriter, req *http.Request, src av.Demuxer, addViewer func(int)) error {
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
	streams, err := src.Streams()
	if err != nil {
		return err
	}
	var m webrtc.MediaEngine
	if err := m.PopulateFromSDP(offer); err != nil {
		return fmt.Errorf("populate from SDP: %w", err)
	}
	api := webrtc.NewAPI(webrtc.WithMediaEngine(m))
	peerConnection, err := api.NewPeerConnection(rtcConf)
	if err != nil {
		return err
	}
	sender := &rtcSender{
		media:  m,
		pc:     peerConnection,
		state:  make(chan webrtc.ICEConnectionState, 1),
		tracks: make([]*rtsp.TrackFramer, len(streams)),
		addr:   req.RemoteAddr,
	}
	sender.pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("[rtc] %s connection state: %s", req.RemoteAddr, state)
		sender.state <- state
	})
	answer, err := sender.setupTracks(streams, offer)
	if err != nil {
		peerConnection.Close()
		return err
	}
	blob, err = json.Marshal(answer)
	if err != nil {
		peerConnection.Close()
		return err
	}
	rw.Write(blob)
	// serve in background
	go func() {
		addViewer(1)
		defer addViewer(-1)
		if err := sender.serve(src); err != nil {
			log.Printf("error: serving rtc to %s: %s", req.RemoteAddr, err)
		}
	}()
	return nil
}

func (s *rtcSender) setupTracks(streams []av.CodecData, offer webrtc.SessionDescription) (*webrtc.SessionDescription, error) {
	ssrc := rand.Uint32()
	for i, stream := range streams {
		codec := s.findCodec(stream)
		if codec == nil {
			return nil, fmt.Errorf("unsupported codec %s for RTSP", stream.Type())
		}
		track, err := s.pc.NewTrack(codec.PayloadType, ssrc, randSeq(), randSeq())
		if err != nil {
			return nil, err
		}
		sconf := webrtc.RtpTransceiverInit{
			Direction: chooseDirection(offer),
		}
		if _, err := s.pc.AddTransceiverFromTrack(track, sconf); err != nil {
			return nil, err
		}
		s.tracks[i] = &rtsp.TrackFramer{
			CodecData: stream,
			Codec:     codec,
			Track:     track,
		}
		ssrc++
	}
	// build answer
	if err := s.pc.SetRemoteDescription(offer); err != nil {
		return nil, err
	}
	answer, err := s.pc.CreateAnswer(nil)
	if err != nil {
		return nil, err
	}
	if err := s.pc.SetLocalDescription(answer); err != nil {
		return nil, err
	}
	return &answer, nil
}

func randSeq() string {
	letters := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]byte, 16)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func (s *rtcSender) serve(src av.Demuxer) error {
	defer s.pc.Close()
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

		packet, err := src.ReadPacket()
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
