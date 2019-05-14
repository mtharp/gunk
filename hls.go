// Copyright © Michael Tharp <gxti@partiallystapled.com>
//
// This file is distributed under the terms of the MIT License.
// See the LICENSE file at the top of this tree or http://opensource.org/licenses/MIT

package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"path"
	"sync/atomic"
	"time"

	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/format/ts"
)

const (
	numChunks = 5
	assumeKI  = 5 * time.Second
)

type hlsChunk struct {
	name       string
	data       []byte
	start, dur time.Duration
}

func (c *hlsChunk) Format() string {
	return fmt.Sprintf("#EXTINF:%.03f,live\n%s\n", c.dur.Seconds(), c.name)
}

func newChunk(data []byte, start, dur time.Duration) *hlsChunk {
	d := make([]byte, len(data))
	copy(d, data)
	return &hlsChunk{
		name:  fmt.Sprintf("%x.ts", time.Now().UnixNano()),
		data:  d,
		start: start,
		dur:   dur,
	}
}

type HLSPublisher struct {
	chunks []*hlsChunk
	last   time.Time
	seq    int64
	state  atomic.Value
}

type hlsState struct {
	playlist []byte
	chunks   []hlsChunk
}

func (p *HLSPublisher) Publish(src av.Demuxer) error {
	streams, err := src.Streams()
	if err != nil {
		return fmt.Errorf("getting streams: %s", err)
	}
	buf := new(bytes.Buffer)
	var chunkMux *ts.Muxer
	var lastKey time.Duration
	for {
		pkt, err := src.ReadPacket()
		if err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("reading stream: %s", err)
		}
		if pkt.IsKeyFrame {
			if chunkMux != nil {
				chunkMux.WriteTrailer()
				chunk := newChunk(buf.Bytes(), pkt.Time, pkt.Time-lastKey)
				p.addChunk(chunk)
			} else {
				chunkMux = ts.NewMuxer(buf)
			}
			buf.Reset()
			chunkMux.WriteHeader(streams)
			lastKey = pkt.Time
		}
		if chunkMux != nil {
			chunkMux.WritePacket(pkt)
		}
	}
	return nil
}

func (p *HLSPublisher) addChunk(chunk *hlsChunk) {
	// shift chunks
	p.chunks = append(p.chunks, chunk)
	if len(p.chunks) > numChunks {
		copy(p.chunks, p.chunks[1:])
		p.chunks = p.chunks[:numChunks]
	}
	p.seq++
	// determine keyframe time
	var maxTime time.Duration
	for _, chunk := range p.chunks {
		if chunk.dur > maxTime {
			maxTime = chunk.dur
		}
	}
	maxTime = maxTime.Round(time.Second)
	// build playlist
	var b bytes.Buffer
	if maxTime == 0 {
		maxTime = assumeKI
	}
	fmt.Fprintf(&b, "#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-TARGETDURATION:%d\n", int(maxTime.Seconds()))
	fmt.Fprintf(&b, "#EXT-X-MEDIA-SEQUENCE:%d\n", p.seq)
	for _, chunk := range p.chunks {
		b.WriteString(chunk.Format())
	}
	// publish
	pubChunks := make([]hlsChunk, len(p.chunks))
	for i, chunk := range p.chunks {
		pubChunks[i] = *chunk
	}
	p.state.Store(hlsState{b.Bytes(), pubChunks})
}

func (p *HLSPublisher) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	state, ok := p.state.Load().(hlsState)
	if !ok {
		http.NotFound(rw, req)
		return
	}
	bn := path.Base(req.URL.Path)
	if bn == "index.m3u8" {
		rw.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		rw.Write(state.playlist)
		return
	}
	for _, chunk := range state.chunks {
		if chunk.name == bn {
			rw.Header().Set("Content-Type", "video/MP2T")
			rw.Write(chunk.data)
			return
		}
	}
	http.NotFound(rw, req)
}
