// Copyright © Michael Tharp <gxti@partiallystapled.com>
//
// This file is distributed under the terms of the MIT License.
// See the LICENSE file at the top of this tree or http://opensource.org/licenses/MIT

package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os/exec"
	"time"

	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/codec/h264parser"
)

const (
	targetWidth  = 400
	grabInterval = 10 * time.Second
)

func grabFrames(channelName string, dm av.Demuxer) error {
	streams, err := dm.Streams()
	if err != nil {
		return err
	}
	vidIdx := -1
	var vidCodec h264parser.CodecData
	for i, s := range streams {
		if s.Type() == av.H264 {
			vidIdx = i
			vidCodec = s.(h264parser.CodecData)
		}
	}
	if vidIdx < 0 {
		return errors.New("no h264 stream found")
	}
	go func() {
		var buf bytes.Buffer
		var keyTime time.Duration
		var lastGrab time.Time
		for {
			pkt, err := dm.ReadPacket()
			if err == io.EOF {
				return
			} else if err != nil {
				log.Println("error: in frame grabber: %s", err)
				return
			}
			if int(pkt.Idx) != vidIdx {
				continue
			}
			if buf.Len() != 0 && (!pkt.IsKeyFrame || pkt.Time != keyTime) {
				if time.Since(lastGrab) >= grabInterval {
					if err := makeFrame(channelName, vidCodec, buf.Bytes()); err != nil {
						log.Println("error: making thumbnail:", err)
					}
					lastGrab = time.Now()
				}
				buf.Reset()
			}
			if !pkt.IsKeyFrame {
				continue
			}
			writeAnnexBPacket(&buf, pkt, vidCodec)
			keyTime = pkt.Time
		}
	}()
	return nil
}

func makeFrame(channelName string, cd h264parser.CodecData, raw []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	height := targetWidth * cd.Height() / cd.Width()
	var jpeg bytes.Buffer
	var errmsg bytes.Buffer
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-loglevel", "warning",
		"-f", "h264",
		"-i", "-",
		"-frames", "1",
		"-s", fmt.Sprintf("%dx%d", targetWidth, height),
		"-f", "singlejpeg", "-")
	cmd.Stdin = bytes.NewReader(raw)
	cmd.Stdout = &jpeg
	cmd.Stderr = &errmsg
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s\n%s", err.Error(), errmsg.String())
	}
	return putThumb(channelName, jpeg.Bytes())
}
