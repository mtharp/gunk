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
	"sync"
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
	mu        sync.Mutex
	chunks    []*hlsChunk
	targetDur time.Duration
	last      time.Time
	seq       int64
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
	p.mu.Lock()
	p.chunks = append(p.chunks, chunk)
	if len(p.chunks) > numChunks {
		copy(p.chunks, p.chunks[1:])
		p.chunks = p.chunks[:numChunks]
	}
	p.seq++
	var maxTime time.Duration
	for _, chunk := range p.chunks {
		if chunk.dur > maxTime {
			maxTime = chunk.dur
		}
	}
	p.targetDur = maxTime.Round(time.Second)
	p.last = time.Now()
	p.mu.Unlock()
}

func (p *HLSPublisher) Last() time.Time {
	p.mu.Lock()
	t := p.last
	p.mu.Unlock()
	return t
}

func (p *HLSPublisher) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	bn := path.Base(req.URL.Path)
	if bn == "index.m3u8" {
		p.servePlaylist(rw, req)
		return
	}
	var chunk *hlsChunk
	p.mu.Lock()
	for _, c := range p.chunks {
		if c.name == bn {
			chunk = c
		}
	}
	p.mu.Unlock()
	if chunk != nil {
		rw.Header().Set("Content-Type", "video/MP2T")
		rw.Write(chunk.data)
	} else {
		http.NotFound(rw, req)
	}
}

func (p *HLSPublisher) servePlaylist(rw http.ResponseWriter, req *http.Request) {
	var b bytes.Buffer
	p.mu.Lock()
	td := p.targetDur
	if td == 0 {
		td = assumeKI
	}
	fmt.Fprintf(&b, "#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-TARGETDURATION:%d\n", int(td.Seconds()))
	fmt.Fprintf(&b, "#EXT-X-MEDIA-SEQUENCE:%d\n", p.seq)
	for _, chunk := range p.chunks {
		b.WriteString(chunk.Format())
	}
	p.mu.Unlock()
	rw.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	rw.Write(b.Bytes())
}
