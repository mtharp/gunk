package irtmp

import (
	"log"
	"net"
	"net/url"

	"eaglesong.dev/gunk/model"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/av/pktque"
	"github.com/nareix/joy4/format/rtmp"
)

type Server struct {
	rtmp.Server
	CheckUser CheckUserFunc
	Publish   PublishFunc
}

type CheckUserFunc func(*url.URL) (model.ChannelAuth, error)
type PublishFunc func(auth model.ChannelAuth, kind, remoteAddr string, src av.Demuxer) error

func (s *Server) ListenAndServe() error {
	s.HandlePublish = s.handlePublish
	return s.Server.ListenAndServe()
}

func (s *Server) handlePublish(conn *rtmp.Conn) {
	defer conn.Close()
	remote := conn.NetConn().RemoteAddr().(*net.TCPAddr).IP.String()
	fm := &pktque.FilterDemuxer{
		Demuxer: conn,
		Filter:  &pktque.FixTime{MakeIncrement: true},
	}
	auth, err := s.CheckUser(conn.URL)
	if err != nil {
		log.Printf("[rtmp] error: %s from %s: %s", conn.URL, remote, err)
		return
	}
	if err := s.Publish(auth, "rtmp", remote, fm); err != nil {
		log.Printf("[rtmp] error: %s from %s: %s", conn.URL, remote, err)
	}
}
