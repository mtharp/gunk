package playrtc

import (
	"strings"

	"github.com/nareix/joy4/av"
	"github.com/pion/sdp/v2"
	"github.com/pion/webrtc/v2"
)

func (s *Sender) findCodec(cd av.CodecData) *webrtc.RTPCodec {
	kind := webrtc.RTPCodecTypeAudio
	if cd.Type().IsVideo() {
		kind = webrtc.RTPCodecTypeVideo
	}
	wantName := strings.ToLower(cd.Type().String())
	codecs := s.media.GetCodecsByKind(kind)
	var chosen *webrtc.RTPCodec
	for _, codec := range codecs {
		if strings.ToLower(codec.Name) == wantName {
			if chosen == nil || chosen.PayloadType < codec.PayloadType {
				chosen = codec
			}
		}
	}
	return chosen
}

// browsers like safari that don't support addTransceiver don't seem to have a way to create recvonly tracks, so try to use sendonly if the browser requested recvonly but fall back to sendrecv if needed
func chooseDirection(offer webrtc.SessionDescription) webrtc.RTPTransceiverDirection {
	var parsed sdp.SessionDescription
	if err := parsed.Unmarshal([]byte(offer.SDP)); err != nil {
		return webrtc.RTPTransceiverDirectionSendrecv
	}
	var recvonly bool
	for _, media := range parsed.MediaDescriptions {
		for _, att := range media.Attributes {
			switch att.Key {
			case "recvonly":
				recvonly = true
			case "sendrecv":
				return webrtc.RTPTransceiverDirectionSendrecv
			}
		}
	}
	if recvonly {
		return webrtc.RTPTransceiverDirectionSendonly
	}
	return webrtc.RTPTransceiverDirectionSendrecv
}
