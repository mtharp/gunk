// Copyright © Michael Tharp <gxti@partiallystapled.com>
//
// This file is distributed under the terms of the MIT License.
// See the LICENSE file at the top of this tree or http://opensource.org/licenses/MIT

package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/kr/pretty"
	"github.com/nareix/joy4/av/pktque"
	"github.com/nareix/joy4/format"
	"github.com/nareix/joy4/format/rtmp"
	"github.com/nareix/joy4/format/ts"
)

const numChunks = 5

type hlsChunk struct {
	name   string
	length time.Duration
	data   []byte
}

func (c *hlsChunk) Format() string {
	return fmt.Sprintf("#EXTINF:%.03f,live\n%s\n", float32(c.length)/float32(time.Second), c.name)
}

func main() {
	format.RegisterAll()
	server := &rtmp.Server{}

	var chunks []*hlsChunk
	var cmu sync.Mutex
	var seq int64
	keyTime := 2 * time.Second

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
		pretty.Println(streams)
		m3u, err := os.Create("/tmp/index.m3u8")
		if err != nil {
			log.Fatalln("error:", err)
		}
		fmt.Fprintln(m3u, "#EXTM3U")
		fmt.Fprintln(m3u, "#EXT-X-VERSION:3")
		fmt.Fprintln(m3u, "#EXT-X-TARGETDURATION:2")
		var chf bytes.Buffer
		var gotKey bool
		keyTime := 2 * time.Second
		var chunkStarts, chunkEnds time.Duration
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
				newChunk := !gotKey
				if gotKey && chunkEnds-pkt.Time < 100*time.Millisecond {
					chm.WriteTrailer()
					data := make([]byte, chf.Len())
					copy(data, chf.Bytes())
					chunk := &hlsChunk{
						name:   fmt.Sprintf("%s-%x.ts", path.Base(conn.URL.Path), time.Now().UnixNano()),
						length: chunkEnds - chunkStarts,
						data:   data,
					}
					log.Println("end  ", chunk.name)
					cmu.Lock()
					if len(chunks) >= numChunks {
						copy(chunks, chunks[1:])
						chunks = chunks[:numChunks]
					}
					chunks = append(chunks, chunk)
					seq++
					cmu.Unlock()
					newChunk = true
				}
				if newChunk {
					chf.Reset()
					chm.SetWriter(&chf)
					chm.WriteHeader(streams)
					chunkStarts = pkt.Time
					chunkEnds = pkt.Time + keyTime
				}
				gotKey = true
			}
			if gotKey {
				chm.WritePacket(pkt)
			}
		}
	}

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		if strings.HasSuffix(req.URL.Path, ".m3u8") {
			var b bytes.Buffer
			fmt.Fprintf(&b, "#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-TARGETDURATION:%d\n", keyTime/time.Second)
			cmu.Lock()
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
				w.Write(chunk.data)
			} else {
				http.NotFound(w, req)
			}
		}
	})

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
