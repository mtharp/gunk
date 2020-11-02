package rtsp

import (
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/textproto"
	"strconv"
	"strings"

	"github.com/nareix/joy4/av"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/sdp/v2"
	"github.com/pion/webrtc/v2"
)

// predefined payload types
var (
	OpusCodec = webrtc.NewRTPOpusCodec(webrtc.DefaultPayloadTypeOpus, 48000)
	H264Codec = webrtc.NewRTPCodec(webrtc.RTPCodecTypeVideo, webrtc.H264, 90000, 0, "profile-level-id=42e01f;level-asymmetry-allowed=1;packetization-mode=1", 102, new(codecs.H264Payloader))
)

type track struct {
	framer *RTPFramer
}

func (c *Conn) handleDescribe(req *Request) error {
	demux, err := c.s.Source(req)
	if err == ErrNotFound {
		return c.WriteResponse(req, 404, nil, nil)
	} else if err != nil {
		return err
	}
	streams, err := demux.Streams()
	if err != nil {
		return err
	}
	hdr := make(textproto.MIMEHeader)
	hdr.Set("Content-Base", req.URL.String())
	hdr.Set("Content-Type", "application/sdp")
	c.ssrc = rand.Uint32()
	c.tracks = make([]*track, len(streams))
	ses, err := sdp.NewJSEPSessionDescription(false)
	if err != nil {
		return err
	}
	var pt uint8 = 96
	for i, stream := range streams {
		var codec *webrtc.RTPCodec
		switch stream.Type() {
		case av.H264:
			codec = H264Codec
		case av.OPUS:
			codec = OpusCodec
		default:
			return fmt.Errorf("unsupported codec %s for RTSP", stream.Type())
		}
		media := &sdp.MediaDescription{
			MediaName: sdp.MediaName{
				Media:  codec.Type.String(),
				Protos: []string{"RTP", "AVP"},
			},
		}
		media.WithCodec(codec.PayloadType, codec.Name, codec.ClockRate, codec.Channels, codec.SDPFmtpLine)
		ses.WithMedia(media)

		packetizer := rtp.NewPacketizer(1400, codec.PayloadType, c.ssrc, codec.Payloader, rtp.NewRandomSequencer(), codec.ClockRate)
		c.tracks[i] = &track{
			framer: &RTPFramer{
				Conn:       c.s.RTPSocket,
				Packetizer: packetizer,
				Codec:      codec,
				CodecData:  stream,
			},
		}
		pt++
	}
	blob, err := ses.Marshal()
	if err != nil {
		return err
	}
	return c.WriteResponse(req, 200, hdr, blob)
}

func (c *Conn) handleSetup(req *Request) error {
	transport := req.Header.Get("Transport")
	words := strings.Split(transport, ";")
	var portRange string
	for _, word := range words {
		if strings.HasPrefix(word, "client_port=") {
			portRange = word[12:]
		}
	}
	if portRange == "" {
		return fmt.Errorf("missing client_port in transport %q", transport)
	}
	destPort, _ := strconv.Atoi(strings.Split(portRange, "-")[0])
	if destPort == 0 {
		return fmt.Errorf("invalid client_port in transport %q", transport)
	}
	srcPort := c.s.RTPSocket.LocalAddr().(*net.UDPAddr).Port
	remoteAddr := c.conn.RemoteAddr().(*net.TCPAddr)
	c.destAddr = &net.UDPAddr{IP: remoteAddr.IP, Port: destPort}
	for _, t := range c.tracks {
		t.framer.Addr = c.destAddr
	}

	transport = fmt.Sprintf("%s;server_port=%d;ssrc=%08X", transport, srcPort, c.ssrc)
	hdr := make(textproto.MIMEHeader)
	hdr.Set("Transport", transport)
	hdr.Set("Session", strconv.FormatUint(uint64(c.ssrc), 10))
	return c.WriteResponse(req, 200, hdr, nil)
}

func (c *Conn) handlePlay(req *Request) error {
	if c.destAddr == nil {
		return errors.New("SETUP not called")
	}
	demux, err := c.s.Source(req)
	if err == ErrNotFound {
		return c.WriteResponse(req, 404, nil, nil)
	} else if err != nil {
		return err
	}
	go func() {
		log.Printf("[rtsp] started sending to %s", c.conn.RemoteAddr())
		defer log.Printf("[rtsp] stopped sending to %s", c.conn.RemoteAddr())
		for c.ctx.Err() == nil {
			pkt, err := demux.ReadPacket()
			if err == io.EOF {
				break
			} else if err != nil {
				log.Printf("error: rtsp %s: reading packet: %s", c.conn.RemoteAddr(), err)
				break
			}
			track := c.tracks[int(pkt.Idx)]
			if track == nil {
				continue
			}
			if err := track.framer.WritePacket(pkt); err != nil {
				if c.ctx.Err() != nil {
					// connection closed
					break
				}
				log.Printf("error: rtsp %s: writing packet: %s", c.conn.RemoteAddr(), err)
				break
			}
		}
	}()
	hdr := make(textproto.MIMEHeader)
	hdr.Set("Session", strconv.FormatUint(uint64(c.ssrc), 10))
	return c.WriteResponse(req, 200, nil, nil)
}
