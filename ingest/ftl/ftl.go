package ftl

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/textproto"
	"runtime"
	"strings"
	"sync"
	"time"

	"eaglesong.dev/gunk/model"
	"github.com/nareix/joy4/av"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Server struct {
	CheckUser CheckUserFunc
	Publish   PublishFunc
	Listener  net.Listener
	RTPSocket net.PacketConn

	RTPAdvertisePort int

	mu        sync.Mutex
	receivers map[string]chan<- []byte
}

type CheckUserFunc func(channelID string, nonce, hmacProvided []byte) (auth model.ChannelAuth, err error)
type PublishFunc func(ctx context.Context, auth model.ChannelAuth, src av.Demuxer) error

func (s *Server) Listen(addr string) (err error) {
	if addr == "" {
		addr = ":8084"
	}
	s.Listener, err = net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s.RTPSocket, err = net.ListenPacket("udp", addr)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) Serve() error {
	go s.serveRTP()
	for {
		conn, err := s.Listener.Accept()
		if err != nil {
			log.Err(err).Msg("error accepting FTL connection")
			time.Sleep(time.Second)
			continue
		}
		l := log.With().Stringer("ftl_ip", conn.RemoteAddr()).Logger()
		c := &Conn{
			s:    s,
			conn: conn,
			log:  l,
		}
		go func() {
			defer func() {
				if r := recover(); r != nil {
					const size = 64 << 10
					buf := make([]byte, size)
					buf = buf[:runtime.Stack(buf, false)]
					l.Error().Interface("error", r).Str("stack", string(buf)).Msg("panic in handler for FTL connection")
				}
				conn.Close()
			}()
			if err := c.serve(); err != nil {
				l.Err(err).Msg("unhandled error in FTL connection")
			}
		}()
	}
}

type connState int

const (
	stateNew connState = iota
	stateUnauth
	stateConfig
	stateLive
)

type Conn struct {
	s      *Server
	conn   net.Conn
	tpc    *textproto.Conn
	ctx    context.Context
	cancel context.CancelFunc
	log    zerolog.Logger

	state connState
	nonce []byte
	auth  model.ChannelAuth

	video, audio       bool
	vcodec, acodec     string
	vpayload, apayload uint8
	vssrc, assrc       uint32
}

func (c *Conn) serve() error {
	c.ctx, c.cancel = context.WithCancel(context.Background())
	defer c.cancel()
	c.tpc = textproto.NewConn(c.conn)
	defer c.tpc.W.Flush()
	authBy := time.Now().Add(30 * time.Second)
	for c.ctx.Err() == nil {
		timeout := 30 * time.Second
		if c.state < stateConfig {
			if time.Now().After(authBy) {
				return errors.New("client didn't auth before deadline")
			}
			timeout = 10 * time.Second
		}
		deadline := time.Now().Add(timeout)
		c.conn.SetReadDeadline(deadline)
		c.conn.SetWriteDeadline(deadline)

		line, err := c.tpc.ReadLine()
		if err != nil {
			if t, ok := err.(timeouter); ok && t.Timeout() {
				return errors.New("timed out waiting for command")
			}
			return err
		} else if line == "" {
			continue
		}
		words := strings.Fields(line)
		switch words[0] {
		case "HMAC":
			err = c.handleHMAC()
		case "CONNECT":
			err = c.handleConnect(words)
		case "DISCONNECT":
			c.log.Info().Msg("source disconnected")
			c.sendOK()
			return nil
		case "PING":
			_, err = c.tpc.W.WriteString("201 PONG.\n")

		case "ProtocolVersion:":
			if len(words) < 2 || words[1] != "0.9" {
				c.badRequest()
				return fmt.Errorf("unsupported protocol version: %s", line)
			}
		case "VendorName:", "VendorVersion:", "VideoHeight:", "VideoWidth:":
			// ignore
		case "Video:", "Audio:":
			err = c.handleEnable(words)
		case "VideoCodec:", "AudioCodec:":
			err = c.handleCodec(words)
		case "VideoPayloadType:", "AudioPayloadType:":
			err = c.handlePT(words)
		case "VideoIngestSSRC:", "AudioIngestSSRC:":
			err = c.handleSSRC(words)

		case ".":
			err = c.handleLive()

		default:
			c.badRequest()
			return fmt.Errorf("unexpected command %q", line)
		}
		if err != nil {
			c.badRequest()
			return err
		}
		if err := c.tpc.W.Flush(); err != nil {
			return err
		}
	}
	return nil
}

func (c *Conn) badRequest() error {
	_, err := c.tpc.W.WriteString("400 Bad Request.\n")
	return err
}

func (c *Conn) sendOK() error {
	_, err := c.tpc.W.WriteString("200 OK.\n")
	return err
}

type timeouter interface {
	Timeout() bool
}
