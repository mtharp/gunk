// Copyright © Michael Tharp <gxti@partiallystapled.com>
//
// This file is distributed under the terms of the MIT License.
// See the LICENSE file at the top of this tree or http://opensource.org/licenses/MIT

package ftl

import (
	"bytes"
	"errors"
	"log"

	"github.com/mtharp/gunk/h264util"
	"github.com/mtharp/gunk/rtsp"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/codec/h264parser"
	"github.com/pion/rtp"
)

type Deframer struct {
	SSRC        uint32
	PayloadType uint8
	ClockRate   uint64
	Parser      Parser

	baseTS  uint64
	lastTS  uint32
	lastSeq uint16
}

type Parser interface {
	ParseFrame([]byte) ([]av.Packet, error)
	CodecData() (av.CodecData, error)
}

func (f *Deframer) Deframe(rp *rtp.Packet) ([]av.Packet, error) {
	seqDelta := rp.SequenceNumber - f.lastSeq
	if seqDelta != 1 {
		log.Printf("seq delta %d", int16(seqDelta))
	}
	f.lastSeq = rp.SequenceNumber

	if rp.Timestamp < f.lastTS && rp.Timestamp < (1<<31) {
		f.baseTS += (1 << 32)
	}
	ts := f.baseTS + uint64(rp.Timestamp)
	f.lastTS = rp.Timestamp
	t := rtsp.FromTS(ts, f.ClockRate)
	packets, err := f.Parser.ParseFrame(rp.Payload)
	for i := range packets {
		packets[i].Time = t
	}

	return packets, err
}

type H264Parser struct {
	sps, pps []byte
	pkt      av.Packet
	fbuf     bytes.Buffer
}

func (p *H264Parser) ParseFrame(packet []byte) ([]av.Packet, error) {
	if len(packet) < 2 {
		return nil, errors.New("rtp: h264 packet too short")
	}
	naluType := packet[0] & 0x1f
	switch {
	case naluType == 7:
		p.sps = packet
		return nil, nil
	case naluType == 8:
		p.pps = packet
		return nil, nil

	case naluType == 28: // FU-A
		/*
			0                   1                   2                   3
			0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
			+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
			| FU indicator  |   FU header   |                               |
			+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+                               |
			|                                                               |
			|                         FU payload                            |
			|                                                               |
			|                               +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
			|                               :...OPTIONAL RTP padding        |
			+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
			Figure 14.  RTP payload format for FU-A

			The FU indicator octet has the following format:
			+---------------+
			|0|1|2|3|4|5|6|7|
			+-+-+-+-+-+-+-+-+
			|F|NRI|  Type   |
			+---------------+

			The FU header has the following format:
			+---------------+
			|0|1|2|3|4|5|6|7|
			+-+-+-+-+-+-+-+-+
			|S|E|R|  Type   |
			+---------------+

			S: 1 bit
			When set to one, the Start bit indicates the start of a fragmented
			NAL unit.  When the following FU payload is not the start of a
			fragmented NAL unit payload, the Start bit is set to zero.

			E: 1 bit
			When set to one, the End bit indicates the end of a fragmented NAL
			unit, i.e., the last byte of the payload is also the last byte of
			the fragmented NAL unit.  When the following FU payload is not the
			last fragment of a fragmented NAL unit, the End bit is set to
			zero.

			R: 1 bit
			The Reserved bit MUST be equal to 0 and MUST be ignored by the
			receiver.

			Type: 5 bits
			The NAL unit payload type as defined in table 7-1 of [1].
		*/
		fuIndicator, fuHeader := packet[0], packet[1]
		if fuHeader&0x80 != 0 {
			p.fbuf.Reset()
			p.fbuf.WriteByte(fuIndicator&0xe0 | fuHeader&0x1f)
		}
		if p.fbuf.Len() != 0 {
			p.fbuf.Write(packet[2:])
			if fuHeader&0x40 != 0 {
				defer p.fbuf.Reset()
				return p.ParseFrame(p.fbuf.Bytes())
			}
		}
		// accumulate more fragments
		return nil, nil

	case naluType == 24: // STAP-A
		/*
			0                   1                   2                   3
			0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
			+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
			|                          RTP Header                           |
			+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
			|STAP-A NAL HDR |         NALU 1 Size           | NALU 1 HDR    |
			+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
			|                         NALU 1 Data                           |
			:                                                               :
			+               +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
			|               | NALU 2 Size                   | NALU 2 HDR    |
			+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
			|                         NALU 2 Data                           |
			:                                                               :
			|                               +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
			|                               :...OPTIONAL RTP padding        |
			+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

			Figure 7.  An example of an RTP packet including an STAP-A
			containing two single-time aggregation units
		*/
		packet = packet[1:]
		var packets []av.Packet
		for len(packet) >= 2 {
			size := int(packet[0])<<8 | int(packet[1])
			if size+2 > len(packet) {
				break
			}
			p2, err := p.ParseFrame(packet[2 : 2+size])
			if err != nil {
				return nil, err
			}
			packets = append(packets, p2...)
			packet = packet[2+size:]
		}
		return packets, nil

	default:
		d := h264util.NALUToAVCC(packet)
		pkt := av.Packet{Data: d}
		if naluType == 5 {
			pkt.IsKeyFrame = true
		}
		return []av.Packet{pkt}, nil
	}
}

func (p *H264Parser) CodecData() (av.CodecData, error) {
	if p.sps == nil || p.pps == nil {
		return nil, nil
	}
	return h264parser.NewCodecDataFromSPSAndPPS(p.sps, p.pps)
}

type NullParser struct {
	Info av.CodecData
}

func (NullParser) ParseFrame(packet []byte) ([]av.Packet, error) {
	return []av.Packet{{Data: packet}}, nil
}

func (p NullParser) CodecData() (av.CodecData, error) {
	return p.Info, nil
}
