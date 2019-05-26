// Copyright © Michael Tharp <gxti@partiallystapled.com>
//
// This file is distributed under the terms of the MIT License.
// See the LICENSE file at the top of this tree or http://opensource.org/licenses/MIT

package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

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

func (s *gunkServer) listChannels() ([]*channelInfo, error) {
	infos, err := listChannels()
	if err != nil {
		return nil, err
	}
	for _, info := range infos {
		if err := s.populateChannel(info); err != nil {
			return nil, err
		}
	}
	s.mu.Lock()
	for _, info := range infos {
		if s.channels[info.Name] != nil {
			info.Live = true
		}
	}
	s.mu.Unlock()
	return infos, nil
}

func (s *gunkServer) populateChannel(info *channelInfo) error {
	u, err := s.router.Get("thumbs").URL("channel", info.Name, "timestamp", strconv.FormatInt(info.Last, 10))
	if err != nil {
		return err
	}
	info.Thumb = u.String()
	return nil
}

func (s *gunkServer) handleChannels(rw http.ResponseWriter, req *http.Request) {
	infos, err := s.listChannels()
	if err != nil {
		log.Printf("error: listing channels: %s", err)
		http.Error(rw, "", 500)
	}
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
	setImmutable(rw)
	rw.Header().Set("Content-Type", "image/jpeg")
	rw.Write(jpeg)
}
