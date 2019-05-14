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

	"github.com/mtharp/gunk/opus"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/av/pubsub"
	"github.com/nareix/joy4/codec/h264parser"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v2"
	"github.com/pion/webrtc/v2/pkg/media"
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

var (
	opusCodec = webrtc.NewRTPOpusCodec(webrtc.DefaultPayloadTypeOpus, 48000)
	h264Codec = webrtc.NewRTPCodec(webrtc.RTPCodecTypeVideo, webrtc.H264, 90000, 0, "profile-level-id=42e01f;level-asymmetry-allowed=1;packetization-mode=1", 102, new(codecs.H264Payloader))
)

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
	var hasAudio bool
	for _, s := range streams {
		if s.Type().IsAudio() {
			if s.Type() != opus.OPUS {
				return fmt.Errorf("unsupported audio codec %s", s.Type())
			}
			hasAudio = true
		} else {
			if s.Type() != av.H264 {
				return fmt.Errorf("unsupported video codec %s", s.Type())
			}
		}
	}
	var m webrtc.MediaEngine
	m.RegisterCodec(opusCodec)
	m.RegisterCodec(h264Codec)
	api := webrtc.NewAPI(webrtc.WithMediaEngine(m))
	peerConnection, err := api.NewPeerConnection(rtcConf)
	if err != nil {
		return err
	}
	vtrack, err := peerConnection.NewTrack(h264Codec.PayloadType, rand.Uint32(), "video", "video")
	if err != nil {
		peerConnection.Close()
		return err
	}
	if _, err := peerConnection.AddTrack(vtrack); err != nil {
		peerConnection.Close()
		return err
	}
	var atrack *webrtc.Track
	if hasAudio {
		atrack, err = peerConnection.NewTrack(opusCodec.PayloadType, rand.Uint32(), "audio", "audio")
		if err != nil {
			return err
		}
		if _, err := peerConnection.AddTrack(atrack); err != nil {
			return err
		}
	}
	// build answer
	remote := req.RemoteAddr
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
	// serve in background
	go func() {
		if err := serveRTC(peerConnection, vtrack, atrack, dm, remote); err != nil {
			log.Printf("error: serving rtc to %s: %s", remote, err)
		}
	}()
	return nil
}

func serveRTC(pc *webrtc.PeerConnection, vtrack, atrack *webrtc.Track, src av.Demuxer, remote string) error {
	defer pc.Close()
	stateCh := make(chan bool, 1)
	pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("[rtc] %s connection state: %s", remote, state)
		stateCh <- state == webrtc.ICEConnectionStateConnected
	})
	streams, _ := src.Streams()
	aidx := -1
	vidx := -1
	var vcodec h264parser.CodecData
	for i, s := range streams {
		if s.Type().IsVideo() {
			if s.Type() != av.H264 {
				return errors.New("unsupported video codec")
			}
			vidx = i
			vcodec = s.(h264parser.CodecData)
		} else if s.Type().IsAudio() {
			aidx = i
		}
	}
	if vidx < 0 {
		return errors.New("no video in stream")
	}

	t := time.NewTimer(rtcIdleTime)
	var connected bool
	var buf bytes.Buffer
	var tsprev, asprev uint64
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
		switch int(packet.Idx) {
		case vidx:
			// convert timestamp to 90khz
			hi, lo := bits.Mul64(uint64(packet.Time), 90000)
			ts, _ := bits.Div64(hi, lo, uint64(time.Second))
			d := ts - tsprev
			tsprev = ts
			// convert NALUs to Annex B
			buf.Reset()
			writeAnnexBPacket(&buf, packet, vcodec)
			if err := vtrack.WriteSample(media.Sample{
				Data:    buf.Bytes(),
				Samples: uint32(d),
			}); err != nil {
				if time.Since(lastWarn) < 10*time.Second {
					log.Printf("[rtc] writing to %s: %s", remote, err)
					lastWarn = time.Now()
				}
			}
		case aidx:
			hi, lo := bits.Mul64(uint64(packet.Time), 48000)
			ts, _ := bits.Div64(hi, lo, uint64(time.Second))
			d := ts - asprev
			asprev = ts
			if err := atrack.WriteSample(media.Sample{
				Data:    packet.Data,
				Samples: uint32(d),
			}); err != nil {
				if time.Since(lastWarn) < 10*time.Second {
					log.Printf("[rtc] writing to %s: %s", remote, err)
					lastWarn = time.Now()
				}
			}
		}
	}
	return nil
}
