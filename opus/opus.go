// Copyright © Michael Tharp <gxti@partiallystapled.com>
//
// This file is distributed under the terms of the MIT License.
// See the LICENSE file at the top of this tree or http://opensource.org/licenses/MIT

package opus

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/av/pubsub"
	"github.com/nareix/joy4/format/aac"
	"golang.org/x/sync/errgroup"
	"layeh.com/gopus"
)

var OPUS = av.MakeAudioCodecType(344444)

type CodecData struct {
	channels int
}

func (d CodecData) Type() av.CodecType {
	return OPUS
}

func (d CodecData) SampleRate() int {
	return 48000
}

func (d CodecData) ChannelLayout() av.ChannelLayout {
	switch d.channels {
	case 1:
		return av.CH_MONO
	case 2:
		return av.CH_STEREO
	default:
		panic("not implemented")
	}
}

func (d CodecData) SampleFormat() av.SampleFormat {
	return av.S16
}

func (d CodecData) PacketDuration(pkt []byte) (time.Duration, error) {
	if len(pkt) < 1 {
		return 0, errors.New("empty opus packet")
	}
	toc := pkt[0]
	config := toc >> 3
	//stereo := (toc & 0x4) != 0
	code := toc & 0x3
	numFr := 0
	switch code {
	case 0:
		// one frame
		if len(pkt) > 1 {
			numFr = 1
		}
	case 1, 2:
		// two frames
		if len(pkt) > 2 {
			numFr = 2
		}
	case 3:
		// N frames
		if len(pkt) < 2 {
			return 0, errors.New("invalid opus packet")
		}
		numFr = int(pkt[1] & 0x3f)
	}
	return time.Duration(numFr) * opusFrameTimes[config], nil
}

// Convert the audio track from src to opus and write the result to dest.
// Video tracks are copied as-is.
func Convert(src av.Demuxer, dest *pubsub.Queue, bitrate int) error {
	streams, err := src.Streams()
	if err != nil {
		return err
	}
	aidx := -1
	var asrcCodec av.AudioCodecData
	newStreams := make([]av.CodecData, len(streams))
	for i, s := range streams {
		if s.Type().IsAudio() {
			aidx = i
			if s.Type() != av.AAC {
				return fmt.Errorf("unsupported audio codec %s", s.Type())
			}
			asrcCodec = s.(av.AudioCodecData)
		} else {
			newStreams[i] = s
		}
	}
	if aidx < 0 {
		return errors.New("no audio stream found")
	}
	channels := asrcCodec.ChannelLayout().Count()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-loglevel", "warning",
		"-f", "aac",
		"-i", "-",
		"-f", "s16le",
		"-ar", "48000",
		"-",
	)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	defer stdin.Close()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	defer stdout.Close()
	//var errmsg bytes.Buffer
	//cmd.Stderr = &errmsg
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}

	// propagate opus header to output
	newStreams[aidx] = &CodecData{channels: channels}
	if err := dest.WriteHeader(newStreams); err != nil {
		return err
	}

	eg, ctx := errgroup.WithContext(ctx)
	// remux audio and send to ffmpeg
	eg.Go(func() error {
		asrcMux := aac.NewMuxer(stdin)
		defer stdin.Close()
		if err := asrcMux.WriteHeader([]av.CodecData{asrcCodec}); err != nil {
			return err
		}
		for ctx.Err() == nil {
			pkt, err := src.ReadPacket()
			if err == io.EOF {
				break
			} else if err != nil {
				return err
			}
			if int(pkt.Idx) == aidx {
				if err := asrcMux.WritePacket(pkt); err != nil {
					return err
				}
			} else {
				if err := dest.WritePacket(pkt); err != nil {
					return err
				}
			}
		}
		return nil
	})
	// read PCM from ffmpeg, encode and mux
	eg.Go(func() error {
		sampleRate := 48000
		packetLength := 20 * time.Millisecond
		samplesPerPacket := int(time.Duration(sampleRate) * packetLength / time.Second)
		encoder, err := gopus.NewEncoder(sampleRate, channels, gopus.Audio)
		if err != nil {
			return err
		}
		encoder.SetBitrate(bitrate)
		sbuf := make([]byte, samplesPerPacket*channels*2)
		samples := make([]int16, samplesPerPacket*channels)
		var ts time.Duration
		for ctx.Err() == nil {
			if _, err := io.ReadFull(stdout, sbuf); err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			} else if err != nil {
				return err
			}
			for i := range samples {
				samples[i] = int16(sbuf[2*i]) | int16(sbuf[2*i+1])<<8
			}
			encoded, err := encoder.Encode(samples, samplesPerPacket, 1200)
			if err != nil {
				return err
			}
			pkt := av.Packet{
				Idx:  int8(aidx),
				Data: encoded,
				Time: ts,
			}
			if err := dest.WritePacket(pkt); err != nil {
				return err
			}
			ts += packetLength
		}
		return nil
	})
	if err := eg.Wait(); err != nil {
		// ensure ffmpeg is stopped and waited on
		cancel()
		cmd.Wait()
		return err
	}
	return cmd.Wait()
}

var opusFrameTimes = []time.Duration{
	// SILK NB
	10 * time.Millisecond,
	20 * time.Millisecond,
	40 * time.Millisecond,
	60 * time.Millisecond,
	// SILK MB
	10 * time.Millisecond,
	20 * time.Millisecond,
	40 * time.Millisecond,
	60 * time.Millisecond,
	// SILK WB
	10 * time.Millisecond,
	20 * time.Millisecond,
	40 * time.Millisecond,
	60 * time.Millisecond,
	// Hybrid SWB
	10 * time.Millisecond,
	20 * time.Millisecond,
	// Hybrid FB
	10 * time.Millisecond,
	20 * time.Millisecond,
	// CELT NB
	2500 * time.Microsecond,
	5 * time.Millisecond,
	10 * time.Millisecond,
	20 * time.Millisecond,
	// CELT WB
	2500 * time.Microsecond,
	5 * time.Millisecond,
	10 * time.Millisecond,
	20 * time.Millisecond,
	// CELT SWB
	2500 * time.Microsecond,
	5 * time.Millisecond,
	10 * time.Millisecond,
	20 * time.Millisecond,
	// CELT FB
	2500 * time.Microsecond,
	5 * time.Millisecond,
	10 * time.Millisecond,
	20 * time.Millisecond,
}
