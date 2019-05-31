// Copyright © Michael Tharp <gxti@partiallystapled.com>
//
// This file is distributed under the terms of the MIT License.
// See the LICENSE file at the top of this tree or http://opensource.org/licenses/MIT

package rtsp

import (
	"bytes"
	"math/bits"
	"net"
	"time"

	"github.com/mtharp/gunk/h264util"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/codec/h264parser"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v2"
	"github.com/pion/webrtc/v2/pkg/media"
)

// ToTS converts a duration to a number of clock ticks at a rate of clockRate
func ToTS(t time.Duration, clockRate uint64) uint64 {
	hi, lo := bits.Mul64(uint64(t), clockRate)
	ts, _ := bits.Div64(hi, lo, uint64(time.Second))
	return ts
}

// FromTS converts a tick timestamp to a duration
func FromTS(ts, clockRate uint64) time.Duration {
	hi, lo := bits.Mul64(uint64(ts), uint64(time.Second))
	t, _ := bits.Div64(hi, lo, clockRate)
	return time.Duration(t)
}

type framer struct {
	ts  uint64
	got bool
	buf bytes.Buffer
}

func (f *framer) delta(t time.Duration, rate uint64) uint32 {
	ts := ToTS(t, rate)
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
