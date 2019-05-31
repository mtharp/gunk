// Copyright © Michael Tharp <gxti@partiallystapled.com>
//
// This file is distributed under the terms of the MIT License.
// See the LICENSE file at the top of this tree or http://opensource.org/licenses/MIT

package ftl

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/textproto"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/nareix/joy4/av"
)

type Server struct {
	CheckUser CheckUserFunc
	Publish   PublishFunc
	RTPSocket net.PacketConn

	RTPAdvertisePort int

	mu        sync.Mutex
	receivers map[string]chan<- []byte
}

type CheckUserFunc func(channelID string, nonce, hmacProvided []byte) (userID, channelName string, err error)
type PublishFunc func(channelName, userID, remoteAddr string, src av.Demuxer) error

func (s *Server) Serve(lis net.Listener) error {
	go s.serveRTP()
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Println("error: accepting FTL connection:", err)
			time.Sleep(time.Second)
			continue
		}
		c := &Conn{
			s:    s,
			conn: conn,
		}
		go func() {
			defer func() {
				if r := recover(); r != nil {
					const size = 64 << 10
					buf := make([]byte, size)
					buf = buf[:runtime.Stack(buf, false)]
					log.Printf("error: panic in handler for FTL connection %s: %s\n%s\n", conn.RemoteAddr(), r, string(buf))
				}
				conn.Close()
			}()
			if err := c.serve(); err != nil {
				log.Printf("error: handling FTL connection from %s: %s", conn.RemoteAddr(), err)
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

	state       connState
	nonce       []byte
	userID      string
	channelName string

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
			log.Printf("[ftl] %s disconnected cleanly", c.conn.RemoteAddr())
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
