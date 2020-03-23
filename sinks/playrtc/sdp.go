package playrtc

import (
	"strconv"
	"strings"

	"eaglesong.dev/gunk/sinks/rtsp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/sdp/v2"
	"github.com/pion/webrtc/v2"
)

// firefox is very picky about payload type numbers, even when everything else matches. so to appease it, parse its offer to figure out what payload type numbers it wants to use.
func chooseCodec(offer string) (h264Codec, opusCodec *webrtc.RTPCodec, err error) {
	var parsed sdp.SessionDescription
	if err := parsed.Unmarshal([]byte(offer)); err != nil {
		return nil, nil, err
	}
	for _, media := range parsed.MediaDescriptions {
		codecNames := make(map[string]string)
		for _, attr := range media.Attributes {
			if attr.Key != "rtpmap" {
				continue
			}
			// 126 H264/90000
			i := strings.IndexRune(attr.Value, ' ')
			j := strings.IndexRune(attr.Value, '/')
			if i < 0 || j < i {
				continue
			}
			payloadType := attr.Value[:i]
			codecName := attr.Value[i+1 : j]
			codecNames[payloadType] = strings.ToLower(codecName)
		}
		for _, attr := range media.Attributes {
			if attr.Key != "fmtp" {
				continue
			}
			// 126 profile-level-id=42e01f;level-asymmetry-allowed=1;packetization-mode=1
			i := strings.IndexRune(attr.Value, ' ')
			if i < 0 {
				continue
			}
			payloadType := attr.Value[:i]
			fmtp := attr.Value[i+1:]
			pti64, err := strconv.ParseUint(payloadType, 10, 8)
			if err != nil {
				continue
			}
			pti := uint8(pti64)
			switch codecNames[payloadType] {
			case "h264":
				if h264Codec == nil || h264Codec.PayloadType > pti {
					h264Codec = webrtc.NewRTPCodec(webrtc.RTPCodecTypeVideo, webrtc.H264, 90000, 0, fmtp, pti, new(codecs.H264Payloader))
				}
			case "opus":
				if opusCodec == nil || opusCodec.PayloadType > pti {
					opusCodec = webrtc.NewRTPOpusCodec(pti, 48000)
				}
			}
		}
	}
	if h264Codec == nil {
		h264Codec = rtsp.H264Codec
	}
	if opusCodec == nil {
		opusCodec = rtsp.OpusCodec
	}
	return
}
