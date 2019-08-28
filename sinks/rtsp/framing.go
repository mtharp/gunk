package rtsp

import (
	"bytes"
	"net"
	"time"

	"eaglesong.dev/gunk/h264util"
	"eaglesong.dev/gunk/internal"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/codec/h264parser"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v2"
	"github.com/pion/webrtc/v2/pkg/media"
)

type framer struct {
	ts  uint64
	got bool
	buf bytes.Buffer
}

func (f *framer) delta(t time.Duration, rate uint64) uint32 {
	ts := internal.ToTS(t, rate)
	var samples uint32
	if f.got {
		samples = uint32(ts - f.ts)
	}
	f.ts = ts
	f.got = true
	return samples
}

func (f *framer) repack(pkt av.Packet, info av.CodecData) []byte {
	switch i := info.(type) {
	case h264parser.CodecData:
		// convert NALUs to Annex B
		f.buf.Reset()
		h264util.WriteAnnexBPacket(&f.buf, pkt, i)
		return f.buf.Bytes()
	}
	return pkt.Data
}

type RTPFramer struct {
	framer
	Conn       net.PacketConn
	Addr       net.Addr
	Packetizer rtp.Packetizer
	Codec      *webrtc.RTPCodec
	CodecData  av.CodecData
}

func (f *RTPFramer) WritePacket(pkt av.Packet) error {
	samples := f.delta(pkt.Time, uint64(f.Codec.ClockRate))
	data := f.repack(pkt, f.CodecData)
	// packetize and send
	pkts := f.Packetizer.Packetize(data, samples)
	for _, pkt := range pkts {
		d, err := pkt.Marshal()
		if err != nil {
			return err
		}
		if _, err := f.Conn.WriteTo(d, f.Addr); err != nil {
			return err
		}
	}
	return nil
}

type TrackFramer struct {
	framer
	CodecData av.CodecData
	Codec     *webrtc.RTPCodec
	Track     *webrtc.Track
}

func (f *TrackFramer) WritePacket(pkt av.Packet) error {
	// convert timestamp back to clock rate
	samples := f.delta(pkt.Time, uint64(f.Codec.ClockRate))
	// convert NALUs to Annex B
	data := f.repack(pkt, f.CodecData)
	return f.Track.WriteSample(media.Sample{
		Data:    data,
		Samples: samples,
	})
}
