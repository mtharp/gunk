// Copyright © Michael Tharp <gxti@partiallystapled.com>
//
// This file is distributed under the terms of the MIT License.
// See the LICENSE file at the top of this tree or http://opensource.org/licenses/MIT

package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nareix/joy4/av/pktque"
	"github.com/nareix/joy4/format"
	"github.com/nareix/joy4/format/rtmp"
	"github.com/nareix/joy4/format/ts"
)

const (
	numChunks = 5
	assumeKI  = 2 * time.Second
)

type hlsChunk struct {
	name string
	// atomic
	dur   uint32
	final uint32
	// locked
	mu      sync.Mutex
	buf     []byte
	readers map[hlsReader]struct{}
}

type hlsReader chan struct{}

func NewChunk(base string, maxKeyTime time.Duration) *hlsChunk {
	return &hlsChunk{
		name:    fmt.Sprintf("%s-%x.ts", base, time.Now().UnixNano()),
		dur:     uint32(maxKeyTime / time.Millisecond),
		readers: make(map[hlsReader]struct{}),
	}
}

func (c *hlsChunk) Format() string {
	dur := float32(atomic.LoadUint32(&c.dur)) / 1000
	return fmt.Sprintf("#EXTINF:%.03f,live\n%s\n", dur, c.name)
}

func (c *hlsChunk) Final() bool {
	return atomic.LoadUint32(&c.final) != 0
}

func (c *hlsChunk) GetFrom(start int) []byte {
	c.mu.Lock()
	d := c.buf[start:]
	c.mu.Unlock()
	return d
}

func (c *hlsChunk) Write(d []byte) (int, error) {
	c.mu.Lock()
	c.buf = append(c.buf, d...)
	defer c.mu.Unlock()
	// wake up readers
	for readerCh := range c.readers {
		// non-blocking write
		select {
		case readerCh <- struct{}{}:
		default:
		}
	}
	return len(d), nil
}

func (c *hlsChunk) Finalize(dur time.Duration) {
	atomic.StoreUint32(&c.dur, uint32(dur/time.Millisecond))
	atomic.StoreUint32(&c.final, 1)
	c.mu.Lock()
	for readerCh := range c.readers {
		close(readerCh)
	}
	c.readers = nil
	c.mu.Unlock()
}

func (c *hlsChunk) WriteTo(ctx context.Context, w io.Writer) error {
	if c.Final() {
		_, err := w.Write(c.buf)
		return err
	}
	readerCh := make(hlsReader, 1)
	c.mu.Lock()
	c.readers[readerCh] = struct{}{}
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		delete(c.readers, readerCh)
		c.mu.Unlock()
	}()
	var pos int
	for ctx.Err() == nil {
		d := c.GetFrom(pos)
		if len(d) != 0 {
			if _, err := w.Write(d); err != nil {
				return err
			}
			log.Printf("reader got %d bytes", len(d))
			pos += len(d)
		}
		select {
		case <-ctx.Done():
			log.Printf("reader went away")
			return ctx.Err()
		case _, ok := <-readerCh:
			if !ok {
				// final
				log.Printf("reader saw EOF")
				return nil
			}
		}
	}
	return ctx.Err()
}

func main() {
	format.RegisterAll()
	server := &rtmp.Server{}

	var chunks []*hlsChunk
	var cmu sync.Mutex
	var seq uint64
	var targetDur int

	server.HandlePublish = func(conn *rtmp.Conn) {
		fm := &pktque.FilterDemuxer{
			Demuxer: conn,
			Filter:  &pktque.FixTime{MakeIncrement: true},
		}
		streams, err := fm.Streams()
		if err != nil {
			log.Println("error: getting streams from publish to %s: %s", conn.URL, err)
			return
		}
		base := path.Base(conn.URL.Path)
		var chunkEnds time.Duration
		var chunk *hlsChunk
		var keyTimes [numChunks]time.Duration
		var keyNum uint
		var lastKey time.Duration
		chm := ts.NewMuxer(nil)
		for {
			pkt, err := fm.ReadPacket()
			if err == io.EOF {
				break
			} else if err != nil {
				log.Println("error: reading stream %s: %s", conn.URL, err)
				break
			}
			if pkt.IsKeyFrame {
				log.Printf("keyframe %d %s %s %d", pkt.Idx, pkt.CompositionTime, pkt.Time, len(pkt.Data))
				maxKeyTime := assumeKI
				thisKeyTime := assumeKI
				if lastKey != 0 {
					thisKeyTime = pkt.Time - lastKey
					keyTimes[keyNum] = thisKeyTime
					keyNum = (keyNum + 1) % numChunks
					maxKeyTime = 0
					for _, t := range keyTimes {
						if t > maxKeyTime {
							maxKeyTime = t
						}
					}
				}
				lastKey = pkt.Time
				if chunk != nil && chunkEnds-pkt.Time < 100*time.Millisecond {
					chm.WriteTrailer()
					log.Println("end  ", chunk.name)
					chunk.Finalize(thisKeyTime)
					chunk = nil
				}
				if chunk == nil {
					chunk = NewChunk(base, maxKeyTime)
					chm.SetWriter(chunk)
					chm.WriteHeader(streams)
					chunkEnds = pkt.Time + maxKeyTime
					cmu.Lock()
					if len(chunks) >= numChunks {
						copy(chunks, chunks[1:])
						chunks = chunks[:numChunks]
					}
					chunks = append(chunks, chunk)
					seq++
					targetDur = int(maxKeyTime.Round(time.Second) / time.Second)
					cmu.Unlock()
				}
			}
			if chunk != nil {
				chm.WritePacket(pkt)
			}
		}
	}

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/" {
			http.ServeFile(w, req, "./index.html")
		} else if strings.HasSuffix(req.URL.Path, ".m3u8") {
			var b bytes.Buffer
			cmu.Lock()
			fmt.Fprintf(&b, "#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-TARGETDURATION:%d\n", targetDur)
			fmt.Fprintf(&b, "#EXT-X-MEDIA-SEQUENCE:%d\n", seq)
			for _, chunk := range chunks {
				b.WriteString(chunk.Format())
			}
			cmu.Unlock()
			w.Write(b.Bytes())
		} else {
			var chunk *hlsChunk
			cmu.Lock()
			for _, c := range chunks {
				if path.Base(req.URL.Path) == c.name {
					chunk = c
					break
				}
			}
			cmu.Unlock()
			if chunk != nil {
				if err := chunk.WriteTo(req.Context(), w); err != nil {
					log.Printf("error: writing chunk %s to client %s: %s", chunk.name, req.RemoteAddr, err)
				}
			} else {
				http.NotFound(w, req)
			}
		}
	})

	http.Handle("/node_modules/", http.FileServer(http.Dir(".")))

	go http.ListenAndServe(":8009", nil)

	//	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	//		l.RLock()
	//		ch := channels[r.URL.Path]
	//		l.RUnlock()
	//
	//		if ch != nil {
	//			//w.Header().Set("Content-Type", "video/x-flv")
	//			//w.Header().Set("Transfer-Encoding", "chunked")
	//			w.Header().Set("Access-Control-Allow-Origin", "*")
	//			w.WriteHeader(200)
	//			//flusher := w.(http.Flusher)
	//			//flusher.Flush()
	//			//muxer := flv.NewMuxerWriteFlusher(writeFlusher{httpflusher: flusher, Writer: w})
	//			muxer := ts.NewMuxer(w)
	//			cursor := ch.que.Latest()
	//
	//			avutil.CopyFile(muxer, cursor)
	//		} else {
	//			http.NotFound(w, r)
	//		}
	//	})
	//
	//	go http.ListenAndServe(":8089", nil)

	server.ListenAndServe()

	// ffmpeg -re -i movie.flv -c copy -f flv rtmp://localhost/movie
	// ffmpeg -f avfoundation -i "0:0" .... -f flv rtmp://localhost/screen
	// ffplay http://localhost:8089/movie
	// ffplay http://localhost:8089/screen
}
