// Copyright © Michael Tharp <gxti@partiallystapled.com>
//
// This file is distributed under the terms of the MIT License.
// See the LICENSE file at the top of this tree or http://opensource.org/licenses/MIT

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/format/ts"
	"github.com/pkg/errors"
)

const (
	numChunks = 20
	assumeKI  = 5 * time.Second
)

var bufioPool sync.Pool

type hlsChunk struct {
	// live
	mu     sync.Mutex
	cond   *sync.Cond
	chunks [][]byte
	dur    time.Duration
	final  bool
	// fixed at creation
	start time.Duration
	name  string
	// flushed state
	f         *os.File
	recycle   *os.File
	size      int64
	destroyed bool
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
	go func() {
		if err := c.flush(); err != nil {
			log.Println("error: flushing segment to disk:", err)
		}
	}()
	return nil
}

func (c *hlsChunk) flush() error {
	c.mu.Lock()
	f := c.recycle
	if f == nil {
		c.mu.Unlock()
		var err error
		f, err = ioutil.TempFile("", "")
		if err != nil {
			return errors.Wrap(err, "tempfile")
		}
		if err := os.Remove(f.Name()); err != nil {
			f.Close()
			return errors.Wrap(err, "unlink")
		}
	} else {
		c.recycle = nil
		c.mu.Unlock()
		if _, err := f.Seek(0, 0); err != nil {
			f.Close()
			return errors.Wrap(err, "rewind")
		}
	}
	b, _ := bufioPool.Get().(*bufio.Writer)
	if b == nil {
		b = bufio.NewWriter(f)
	} else {
		b.Reset(f)
	}
	for _, chunk := range c.chunks {
		if _, err := b.Write(chunk); err != nil {
			f.Close()
			return errors.Wrap(err, "write")
		}
		c.size += int64(len(chunk))
	}
	if err := b.Flush(); err != nil {
		f.Close()
		return errors.Wrap(err, "flush")
	}
	b.Reset(nil)
	bufioPool.Put(b)
	c.mu.Lock()
	if c.destroyed {
		// oops, too late
		f.Close()
	} else {
		c.f = f
		c.chunks = nil
	}
	c.mu.Unlock()
	return nil
}

func (c *hlsChunk) Destroy() (recycle *os.File) {
	c.mu.Lock()
	recycle = c.f
	c.f = nil
	c.destroyed = true
	if c.recycle != nil {
		// recycled file didn't get used from last time
		c.recycle.Close()
		c.recycle = nil
	}
	c.mu.Unlock()
	return
}

func (c *hlsChunk) Format() string {
	return fmt.Sprintf("#EXTINF:%.03f,live\n%s\n", c.dur.Seconds(), c.name)
}

func (c *hlsChunk) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Set("Cache-Control", "max-age=600, public")
	rw.Header().Set("Content-Type", "video/MP2T")
	flusher, _ := rw.(http.Flusher)
	c.mu.Lock()
	if c.f != nil {
		c.mu.Unlock()
		rw.Header().Set("Content-Length", strconv.FormatInt(c.size, 10))
		io.Copy(rw, io.NewSectionReader(c.f, 0, c.size))
		return
	}
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
		chunk.recycle = p.chunks[0].Destroy()
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
