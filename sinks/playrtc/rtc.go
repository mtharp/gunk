package playrtc

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"eaglesong.dev/gunk/sinks/rtsp"
	"eaglesong.dev/gunk/transcode/opus"
	"github.com/nareix/joy4/av"
	"github.com/pion/rtp/codecs"
	"github.com/pion/sdp"
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
	h264Codec, err := chooseCodec(offer.SDP)
	if err != nil {
		log.Printf("unable to determine h264 codec attributes: %s", err)
		http.Error(rw, "invalid offer", 400)
		return nil
	}
	m.RegisterCodec(h264Codec)
	m.RegisterCodec(rtsp.OpusCodec)
	api := webrtc.NewAPI(webrtc.WithMediaEngine(m))
	peerConnection, err := api.NewPeerConnection(rtcConf)
	if err != nil {
		return err
	}
	sender := &rtcSender{
		pc:     peerConnection,
		state:  make(chan webrtc.ICEConnectionState, 1),
		tracks: make([]*rtsp.TrackFramer, len(streams)),
		addr:   req.RemoteAddr,
	}
	sender.pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("[rtc] %s connection state: %s", req.RemoteAddr, state)
		sender.state <- state
	})
	answer, err := sender.setupTracks(streams, offer, h264Codec)
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

func (s *rtcSender) setupTracks(streams []av.CodecData, offer webrtc.SessionDescription, h264Codec *webrtc.RTPCodec) (*webrtc.SessionDescription, error) {
	ssrc := rand.Uint32()
	for i, stream := range streams {
		var codec *webrtc.RTPCodec
		switch stream.Type() {
		case av.H264:
			codec = h264Codec
		case opus.OPUS:
			codec = rtsp.OpusCodec
		default:
			return nil, fmt.Errorf("unsupported codec %s for RTSP", stream.Type())
		}
		name := codec.Type.String()
		track, err := s.pc.NewTrack(codec.PayloadType, ssrc, name, name)
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

// firefox is very picky about payload type numbers, even when everything else matches. so to appease it, parse its offer to figure out what payload type numbers it wants to use.
func chooseCodec(offer string) (*webrtc.RTPCodec, error) {
	var parsed sdp.SessionDescription
	if err := parsed.Unmarshal(offer); err != nil {
		return nil, err
	}
	var candidate *webrtc.RTPCodec
	for _, media := range parsed.MediaDescriptions {
		h264Types := make(map[string]struct{})
		for _, attr := range media.Attributes {
			if attr.Key != "rtpmap" {
				continue
			}
			i := strings.IndexRune(attr.Value, ' ')
			j := strings.IndexRune(attr.Value, '/')
			if j < 0 || j < i {
				continue
			}
			payloadType := attr.Value[:i]
			codecName := attr.Value[i+1 : j]
			if codecName == "H264" {
				h264Types[payloadType] = struct{}{}
			}
		}
		if len(h264Types) == 0 {
			continue
		}
		for _, attr := range media.Attributes {
			if attr.Key != "fmtp" {
				continue
			}
			i := strings.IndexRune(attr.Value, ' ')
			if i < 0 {
				continue
			}
			payloadType := attr.Value[:i]
			if _, ok := h264Types[payloadType]; !ok {
				continue
			}
			fmtp := attr.Value[i+1:]
			pti64, err := strconv.ParseUint(payloadType, 10, 8)
			if err != nil {
				continue
			}
			pti := uint8(pti64)
			if candidate == nil || candidate.PayloadType > pti {
				candidate = webrtc.NewRTPCodec(webrtc.RTPCodecTypeVideo, webrtc.H264, 90000, 0, fmtp, pti, new(codecs.H264Payloader))
			}
		}
	}
	if candidate != nil {
		return candidate, nil
	}
	return rtsp.H264Codec, nil
}
