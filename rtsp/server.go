// Copyright © Michael Tharp <gxti@partiallystapled.com>
//
// This file is distributed under the terms of the MIT License.
// See the LICENSE file at the top of this tree or http://opensource.org/licenses/MIT

package rtsp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/textproto"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/nareix/joy4/av"
)

var ErrNotFound = errors.New("stream not found")

type Server struct {
	Source    SourceFunc
	RTPSocket net.PacketConn
}

type SourceFunc func(*Request) (av.Demuxer, error)

func (s *Server) Serve(lis net.Listener) error {
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Println("error: accepting RTSP connection:", err)
			time.Sleep(time.Second)
			continue
		}
		rtspConn := &Conn{
			s:    s,
			conn: conn,
		}
		go rtspConn.serve()
	}
}

type Conn struct {
	s      *Server
	conn   net.Conn
	tpc    *textproto.Conn
	ctx    context.Context
	cancel context.CancelFunc

	ssrc     uint32
	tracks   []*track
	destAddr net.Addr
}

func (c *Conn) serve() {
	defer func() {
		if r := recover(); r != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			log.Printf("error: panic in handler for RTSP connection %s: %s\n%s\n", c.conn.RemoteAddr(), r, string(buf))
		}
		c.conn.Close()
	}()
	c.ctx, c.cancel = context.WithCancel(context.Background())
	defer c.cancel()
	c.tpc = textproto.NewConn(c.conn)
	for {
		if err := c.handleRequest(); err == io.EOF {
			break
		} else if err != nil {
			log.Printf("error: rtsp %s: %s", c.conn.RemoteAddr(), err)
			return
		}
	}
}

var responseText = map[int]string{
	100: "Continue",
	200: "OK",
	400: "Bad Request",
	404: "Not Found",
	405: "Method Not Allowed",
	500: "Internal Server Error",
}

func (c *Conn) WriteResponse(req *Request, code int, headers textproto.MIMEHeader, body []byte) error {
	msg := responseText[code]
	if msg == "" {
		return fmt.Errorf("unknown code %d", code)
	}
	if headers == nil {
		headers = make(textproto.MIMEHeader)
	}
	headers.Set("Cseq", req.CSeq)
	if len(body) != 0 {
		headers.Set("Content-Length", strconv.Itoa(len(body)))
	}
	if err := c.tpc.PrintfLine("RTSP/1.0 %d %s", code, msg); err != nil {
		return err
	}
	for key, values := range headers {
		for _, value := range values {
			if err := c.tpc.PrintfLine("%s: %s", key, value); err != nil {
				return err
			}
		}
	}
	if _, err := c.tpc.W.Write([]byte("\r\n")); err != nil {
		return err
	}
	if _, err := c.tpc.W.Write(body); err != nil {
		return err
	}
	return c.tpc.W.Flush()
}

type Request struct {
	Method string
	URL    *url.URL
	Header textproto.MIMEHeader
	CSeq   string
}

func (c *Conn) handleRequest() error {
	line, err := c.tpc.ReadLine()
	if err != nil {
		return err
	}
	header, err := c.tpc.ReadMIMEHeader()
	if err != nil {
		return err
	}
	words := strings.Fields(line)
	if len(words) != 3 {
		return fmt.Errorf("unexpected request line %q", line)
	}
	u, err := url.Parse(words[1])
	if err != nil {
		return err
	}
	req := &Request{
		Method: words[0],
		URL:    u,
		Header: header,
		CSeq:   header.Get("Cseq"),
	}
	switch req.Method {
	case "OPTIONS":
		resp := make(textproto.MIMEHeader)
		resp.Set("Public", "DESCRIBE, SETUP, PLAY, TEARDOWN")
		err = c.WriteResponse(req, 200, resp, nil)
	case "DESCRIBE":
		err = c.handleDescribe(req)
	case "SETUP":
		err = c.handleSetup(req)
	case "PLAY":
		err = c.handlePlay(req)
	case "TEARDOWN":
		c.cancel()
		return c.WriteResponse(req, 200, nil, nil)
	default:
		return c.WriteResponse(req, 405, nil, nil)
	}
	if err != nil {
		return fmt.Errorf("%s %s: %s", req.Method, req.URL, err)
	}
	return nil
}
