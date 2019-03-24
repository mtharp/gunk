// Copyright © Michael Tharp <gxti@partiallystapled.com>
//
// This file is distributed under the terms of the MIT License.
// See the LICENSE file at the top of this tree or http://opensource.org/licenses/MIT

package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"

	"github.com/gorilla/mux"
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
	if err := handleSDP(rw, req, ch.queue); err != nil {
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
	chNames := []string{}
	s.mu.Lock()
	for chName := range s.channels {
		chNames = append(chNames, chName)
	}
	s.mu.Unlock()
	sort.Strings(chNames)
	blob, _ := json.Marshal(chNames)
	rw.Header().Set("Content-Type", "application/json")
	rw.Write(blob)
}
