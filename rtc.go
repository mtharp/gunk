package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/mtharp/gunk/opus"
	"github.com/mtharp/gunk/rtsp"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/av/pubsub"
	"github.com/pion/webrtc/v2"
)

const rtcIdleTime = 60 * time.Second

var rtcConf = webrtc.Configuration{
	ICEServers: []webrtc.ICEServer{{
		URLs: []string{
			"stun:stun1.l.google.com:19302",
			"stun:stun2.l.google.com:19302",
		},
	}},
}

type rtcSender struct {
	pc     *webrtc.PeerConnection
	tracks []*rtsp.TrackFramer
	addr   string
}

func handleSDP(rw http.ResponseWriter, req *http.Request, queue *pubsub.Queue) error {
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
	dm := queue.Latest()
	streams, err := dm.Streams()
	if err != nil {
		return err
	}
	var m webrtc.MediaEngine
	m.RegisterCodec(rtsp.OpusCodec)
	m.RegisterCodec(rtsp.H264Codec)
	api := webrtc.NewAPI(webrtc.WithMediaEngine(m))
	peerConnection, err := api.NewPeerConnection(rtcConf)
	if err != nil {
		return err
	}
	sender := &rtcSender{
		pc:     peerConnection,
		tracks: make([]*rtsp.TrackFramer, len(streams)),
		addr:   req.RemoteAddr,
	}
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
		if err := sender.serve(dm); err != nil {
			log.Printf("error: serving rtc to %s: %s", req.RemoteAddr, err)
		}
	}()
	return nil
}

func (s *rtcSender) setupTracks(streams []av.CodecData, offer webrtc.SessionDescription) (*webrtc.SessionDescription, error) {
	for i, stream := range streams {
		var codec *webrtc.RTPCodec
		switch stream.Type() {
		case av.H264:
			codec = rtsp.H264Codec
		case opus.OPUS:
			codec = rtsp.OpusCodec
		default:
			return nil, fmt.Errorf("unsupported codec %s for RTSP", stream.Type())
		}
		name := codec.Type.String()
		track, err := s.pc.NewTrack(codec.PayloadType, rand.Uint32(), name, name)
		if err != nil {
			return nil, err
		}
		if _, err := s.pc.AddTrack(track); err != nil {
			return nil, err
		}
		s.tracks[i] = &rtsp.TrackFramer{
			CodecData: stream,
			Codec:     codec,
			Track:     track,
		}
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

func (s *rtcSender) serve(src av.Demuxer) error {
	defer s.pc.Close()
	stateCh := make(chan bool, 1)
	s.pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("[rtc] %s connection state: %s", s.addr, state)
		stateCh <- state == webrtc.ICEConnectionStateConnected
	})
	t := time.NewTimer(rtcIdleTime)
	var connected bool
	var lastWarn time.Time
	for {
		// check if still connected
		select {
		case connected = <-stateCh:
		default:
		}
		for !connected {
			// wait for connection or timeout
			t.Reset(rtcIdleTime)
			select {
			case <-t.C:
				return nil
			case connected = <-stateCh:
			}
		}
		t.Stop()

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
		if err := track.WritePacket(packet); err != nil {
			if time.Since(lastWarn) < 10*time.Second {
				log.Printf("[rtc] writing to %s: %s", s.addr, err)
				lastWarn = time.Now()
			}
		}
	}
	return nil
}
