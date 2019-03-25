// Copyright © Michael Tharp <gxti@partiallystapled.com>
//
// This file is distributed under the terms of the MIT License.
// See the LICENSE file at the top of this tree or http://opensource.org/licenses/MIT

package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx"
	"github.com/nareix/joy4/av/avutil"
	"github.com/nareix/joy4/format/ts"
)

func (s *gunkServer) handleHLS(rw http.ResponseWriter, req *http.Request) {
	chname := mux.Vars(req)["channel"]
	s.mu.Lock()
	ch := s.channels[chname]
	s.mu.Unlock()
	if ch == nil {
		log.Printf("not found: %s", req.URL)
		http.NotFound(rw, req)
		return
	}
	ch.hls.ServeHTTP(rw, req)
}

func (s *gunkServer) handleRTC(rw http.ResponseWriter, req *http.Request) {
	chname := mux.Vars(req)["channel"]
	s.mu.Lock()
	ch := s.channels[chname]
	s.mu.Unlock()
	if ch == nil {
		log.Printf("not found: %s", req.URL)
		http.NotFound(rw, req)
		return
	}
	if err := handleSDP(rw, req, ch.opusq); err != nil {
		log.Printf("error: failed to start webrtc session to %s: %s", req.RemoteAddr, err)
		http.Error(rw, "failed to start webrtc session", 500)
	}
}

func (s *gunkServer) handleTS(rw http.ResponseWriter, req *http.Request) {
	chname := mux.Vars(req)["channel"]
	s.mu.Lock()
	ch := s.channels[chname]
	s.mu.Unlock()
	if ch == nil {
		log.Printf("not found: %s", req.URL)
		http.NotFound(rw, req)
		return
	}
	rw.Header().Set("Content-Type", "video/MP2T")
	rw.Header().Set("Transfer-Encoding", "chunked")
	muxer := ts.NewMuxer(rw)
	avutil.CopyFile(muxer, ch.queue.Latest())
}

func (s *gunkServer) handleChannels(rw http.ResponseWriter, req *http.Request) {
	infos, err := listChannels()
	if err != nil {
		log.Printf("error: listing channels: %s", err)
		http.Error(rw, "", 500)
	}
	s.mu.Lock()
	for i, info := range infos {
		if s.channels[info.Name] != nil {
			infos[i].Live = true
		}
	}
	s.mu.Unlock()
	blob, _ := json.Marshal(infos)
	rw.Header().Set("Content-Type", "application/json")
	rw.Write(blob)
}

func (s *gunkServer) handleThumb(rw http.ResponseWriter, req *http.Request) {
	chname := mux.Vars(req)["channel"]
	jpeg, err := getThumb(chname)
	if err == pgx.ErrNoRows {
		log.Printf("not found: %s", req.URL)
		http.NotFound(rw, req)
		return
	} else if err != nil {
		log.Printf("error: getting thumbnail: %s", err)
		http.Error(rw, "", 500)
		return
	}
	rw.Header().Set("Cache-Control", "max-age=2592000, public, immutable")
	rw.Header().Set("Content-Type", "image/jpeg")
	rw.Write(jpeg)
}
