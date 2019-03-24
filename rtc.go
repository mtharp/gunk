package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/bits"
	"math/rand"
	"net/http"
	"time"

	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/av/pubsub"
	"github.com/nareix/joy4/codec/h264parser"
	"github.com/pions/webrtc"
	"github.com/pions/webrtc/pkg/media"
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

func handleSDP(rw http.ResponseWriter, req *http.Request, queue *pubsub.Queue) error {
	blob, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return err
	}
	var offer webrtc.SessionDescription
	if err := json.Unmarshal(blob, &offer); err != nil {
		http.Error(rw, "invalid offer", 400)
		return nil
	}
	remote := req.RemoteAddr
	peerConnection, err := webrtc.NewPeerConnection(rtcConf)
	if err != nil {
		return err
	}
	vtrack, err := peerConnection.NewTrack(webrtc.DefaultPayloadTypeH264, rand.Uint32(), "video", "video")
	if err != nil {
		peerConnection.Close()
		return err
	}
	if _, err := peerConnection.AddTrack(vtrack); err != nil {
		peerConnection.Close()
		return err
	}
	if err := peerConnection.SetRemoteDescription(offer); err != nil {
		peerConnection.Close()
		return err
	}
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		peerConnection.Close()
		return err
	}
	if err := peerConnection.SetLocalDescription(answer); err != nil {
		peerConnection.Close()
		return err
	}
	blob, err = json.Marshal(answer)
	if err != nil {
		peerConnection.Close()
		return err
	}
	rw.Write(blob)
	go func() {
		if err := serveRTC(peerConnection, vtrack, queue.Latest(), remote); err != nil {
			log.Printf("error: serving rtc to %s: %s", remote, err)
		}
	}()
	return nil
}

func serveRTC(pc *webrtc.PeerConnection, vtrack *webrtc.Track, src av.Demuxer, remote string) error {
	defer pc.Close()
	stateCh := make(chan bool, 1)
	pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("[rtc] %s connection state: %s", remote, state)
		stateCh <- state == webrtc.ICEConnectionStateConnected
	})
	streams, err := src.Streams()
	if err != nil {
		return fmt.Errorf("getting streams: %s", err)
	}
	vidx := -1
	var codec h264parser.CodecData
	for i, s := range streams {
		if s.Type().IsVideo() {
			if s.Type() != av.H264 {
				return errors.New("unsupported video codec")
			}
			vidx = i
			codec = s.(h264parser.CodecData)
		}
	}
	if vidx < 0 {
		return errors.New("no video in stream")
	}

	t := time.NewTimer(rtcIdleTime)
	var connected bool
	var buf bytes.Buffer
	var tsprev uint64
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
		if packet.Idx != int8(vidx) {
			continue
		}
		// convert timestamp to 90khz
		hi, lo := bits.Mul64(uint64(packet.Time), 90000)
		ts, _ := bits.Div64(hi, lo, uint64(time.Second))
		d := ts - tsprev
		tsprev = ts
		// convert NALUs to Annex B
		buf.Reset()
		writeAnnexBPacket(&buf, packet, codec)
		vtrack.WriteSample(media.Sample{
			Data:    buf.Bytes(),
			Samples: uint32(d),
		})
	}
	return nil
}
