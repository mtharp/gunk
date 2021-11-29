package irtmp

import (
	"context"
	"net"
	"net/url"

	"eaglesong.dev/gunk/model"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/av/pktque"
	"github.com/nareix/joy4/format/rtmp"
	"github.com/rs/zerolog/log"
)

type Server struct {
	rtmp.Server
	CheckUser CheckUserFunc
	Publish   PublishFunc
}

type CheckUserFunc func(*url.URL) (model.ChannelAuth, error)
type PublishFunc func(ctx context.Context, auth model.ChannelAuth, src av.Demuxer) error

func (s *Server) ListenAndServe() error {
	s.HandlePublish = s.handlePublish
	return s.Server.ListenAndServe()
}

func (s *Server) handlePublish(conn *rtmp.Conn) {
	defer conn.Close()
	remote := conn.NetConn().RemoteAddr().(*net.TCPAddr).IP.String()
	fm := &pktque.FilterDemuxer{
		Demuxer: conn,
		Filter:  &DeJitter{},
	}
	l := log.With().Str("rtmp_ip", remote).Str("kind", "rtmp").Logger()
	ctx := l.WithContext(context.Background())
	auth, err := s.CheckUser(conn.URL)
	if err != nil {
		l.Err(err).Stringer("rtmp_url", conn.URL).Msg("RTMP auth failed")
		return
	}
	if err := s.Publish(ctx, auth, fm); err != nil {
		l.Err(err).Stringer("rtmp_url", conn.URL).Msg("RTMP publish failed")
	}
}
