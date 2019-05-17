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
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/format/ts"
)

const (
	numChunks = 20
	assumeKI  = 5 * time.Second
)

type hlsChunk struct {
	chunks [][]byte
	mu     sync.Mutex
	cond   *sync.Cond
	name   string
	final  bool
	start  time.Duration
	dur    time.Duration
}

func newChunk(start, initialDur time.Duration) *hlsChunk {
	c := &hlsChunk{
		name:  strconv.FormatInt(time.Now().UnixNano(), 36) + ".ts",
		start: start,
		dur:   initialDur,
	}
	c.cond = &sync.Cond{L: &c.mu}
	return c
}

func (c *hlsChunk) Append(d []byte) {
	if len(d) == 0 {
		return
	}
	buf := make([]byte, len(d))
	copy(buf, d)
	c.mu.Lock()
	c.chunks = append(c.chunks, buf)
	c.mu.Unlock()
	c.cond.Broadcast()
}

func (c *hlsChunk) Close(nextSegment time.Duration) error {
	c.mu.Lock()
	c.dur = nextSegment - c.start
	c.final = true
	c.mu.Unlock()
	c.cond.Broadcast()
	return nil
}

func (c *hlsChunk) Format() string {
	return fmt.Sprintf("#EXTINF:%.03f,live\n%s\n", c.dur.Seconds(), c.name)
}

func (c *hlsChunk) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Set("Cache-Control", "max-age=600, public")
	rw.Header().Set("Content-Type", "video/MP2T")
	flusher, _ := rw.(http.Flusher)
	c.mu.Lock()
	var pos int
	for {
		for pos < len(c.chunks) {
			d := c.chunks[pos]
			pos++
			c.mu.Unlock()
			rw.Write(d)
			c.mu.Lock()
		}
		if c.final {
			break
		}
		if flusher != nil {
			flusher.Flush()
		}
		c.cond.Wait()
	}
	c.mu.Unlock()
}

type HLSPublisher struct {
	chunks []*hlsChunk
	seq    int64
	state  atomic.Value
}

type hlsState struct {
	playlist []byte
	chunks   []*hlsChunk
}

func (p *HLSPublisher) Publish(src av.Demuxer) error {
	streams, err := src.Streams()
	if err != nil {
		return fmt.Errorf("getting streams: %s", err)
	}
	buf := new(bytes.Buffer)
	chunkMux := ts.NewMuxer(buf)
	var chunk *hlsChunk
	for {
		pkt, err := src.ReadPacket()
		if err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("reading stream: %s", err)
		}
		buf.Reset()
		if pkt.IsKeyFrame {
			if chunk != nil {
				chunk.Close(pkt.Time)
			}
			chunk = p.addChunk(pkt.Time)
			chunkMux.WriteHeader(streams)
		}
		if chunk != nil {
			chunkMux.WritePacket(pkt)
			chunk.Append(buf.Bytes())
		}
	}
	return nil
}

func (p *HLSPublisher) targetDuration() time.Duration {
	var maxTime time.Duration
	for _, chunk := range p.chunks {
		if chunk.dur > maxTime {
			maxTime = chunk.dur
		}
	}
	maxTime = maxTime.Round(time.Second)
	if maxTime == 0 {
		maxTime = assumeKI
	}
	return maxTime
}

func (p *HLSPublisher) addChunk(start time.Duration) *hlsChunk {
	initialDur := p.targetDuration()
	chunk := newChunk(start, initialDur)
	// shift chunks
	p.chunks = append(p.chunks, chunk)
	if len(p.chunks) > numChunks {
		copy(p.chunks, p.chunks[1:])
		p.chunks = p.chunks[:numChunks]
	}
	p.seq++
	// build playlist
	var b bytes.Buffer
	fmt.Fprintf(&b, "#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-TARGETDURATION:%d\n", int(initialDur.Seconds()))
	fmt.Fprintf(&b, "#EXT-X-MEDIA-SEQUENCE:%d\n", p.seq)
	for _, chunk := range p.chunks {
		b.WriteString(chunk.Format())
	}
	// publish
	pubChunks := make([]*hlsChunk, len(p.chunks))
	copy(pubChunks, p.chunks)
	p.state.Store(hlsState{b.Bytes(), pubChunks})
	return chunk
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
			chunk.ServeHTTP(rw, req)
			return
		}
	}
	http.NotFound(rw, req)
}
