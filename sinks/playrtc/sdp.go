package playrtc

import (
	"github.com/pion/sdp"
	"github.com/pion/webrtc/v3"
)

// browsers like safari that don't support addTransceiver don't seem to have a way to create recvonly tracks, so try to use sendonly if the browser requested recvonly but fall back to sendrecv if needed
func chooseDirection(offer webrtc.SessionDescription) webrtc.RTPTransceiverDirection {
	var parsed sdp.SessionDescription
	if err := parsed.Unmarshal(offer.SDP); err != nil {
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
