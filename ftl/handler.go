// Copyright © Michael Tharp <gxti@partiallystapled.com>
//
// This file is distributed under the terms of the MIT License.
// See the LICENSE file at the top of this tree or http://opensource.org/licenses/MIT

package ftl

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"

	"github.com/mtharp/gunk/opus"
)

func (c *Conn) handleHMAC() error {
	if c.state > stateUnauth {
		return errors.New("unexpected HMAC after auth complete")
	}
	if c.nonce == nil {
		c.nonce = make([]byte, 64)
		if _, err := io.ReadFull(rand.Reader, c.nonce); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(c.tpc.W, "200 %x\n", c.nonce); err != nil {
		return err
	}
	c.state = stateUnauth
	return nil
}

func (c *Conn) handleConnect(words []string) error {
	if len(words) < 3 {
		return errors.New("invalid CONNECT line")
	} else if c.state != stateUnauth {
		return errors.New("invalid state for CONNECT")
	}
	channelID, digestHex := words[1], words[2]
	if strings.HasPrefix(digestHex, "$") {
		digestHex = digestHex[1:]
	}
	digest, err := hex.DecodeString(digestHex)
	if err != nil {
		return fmt.Errorf("parsing CONNECT: %s", err)
	}
	c.userID, c.channelName, err = c.s.CheckUser(channelID, c.nonce, digest)
	if err != nil {
		return err
	}
	log.Printf("[ftl] authenticated %s as channel %s of user %s", c.conn.RemoteAddr(), c.channelName, c.userID)
	c.state = stateConfig
	return c.sendOK()
}

func (c *Conn) handleEnable(words []string) error {
	if c.state != stateConfig {
		return errors.New("unexpected state")
	}
	if len(words) != 2 || !strings.EqualFold(words[1], "true") {
		return fmt.Errorf("unexpected value: %s", strings.Join(words, " "))
	}
	if words[0][:5] == "Video" {
		c.video = true
	} else {
		c.audio = true
	}
	return nil
}

func (c *Conn) handleCodec(words []string) error {
	if c.state != stateConfig {
		return errors.New("unexpected state")
	}
	if len(words) != 2 {
		return fmt.Errorf("unexpected value: %s", strings.Join(words, " "))
	}
	if words[0][:5] == "Video" {
		c.vcodec = words[1]
	} else {
		c.acodec = words[1]
	}
	return nil
}

func (c *Conn) handlePT(words []string) error {
	if c.state != stateConfig {
		return errors.New("unexpected state")
	}
	if len(words) != 2 {
		return fmt.Errorf("unexpected value: %s", strings.Join(words, " "))
	}
	v, err := strconv.ParseUint(words[1], 10, 8)
	if err != nil {
		return fmt.Errorf("value %q: %s", strings.Join(words, " "), err)
	}
	if words[0][:5] == "Video" {
		c.vpayload = uint8(v)
	} else {
		c.apayload = uint8(v)
	}
	return nil
}

func (c *Conn) handleSSRC(words []string) error {
	if c.state != stateConfig {
		return errors.New("unexpected state")
	}
	if len(words) != 2 {
		return fmt.Errorf("unexpected value: %s", strings.Join(words, " "))
	}
	v, err := strconv.ParseUint(words[1], 10, 32)
	if err != nil {
		return fmt.Errorf("value %q: %s", strings.Join(words, " "), err)
	}
	if words[0][:5] == "Video" {
		c.vssrc = uint32(v)
	} else {
		c.assrc = uint32(v)
	}
	return nil
}

func (c *Conn) handleLive() error {
	if c.state != stateConfig {
		return errors.New("unexpected state")
	}
	c.state = stateLive
	if !c.video || !c.audio ||
		c.vcodec == "" || c.acodec == "" ||
		c.vpayload == 0 || c.apayload == 0 ||
		c.vssrc == 0 || c.assrc == 0 {
		return errors.New("missing parameter")
	}
	// setup codecs
	var vdeframer, adeframer *Deframer
	switch c.vcodec {
	case "H264":
		vdeframer = &Deframer{
			SSRC:        c.vssrc,
			PayloadType: c.vpayload,
			ClockRate:   90000,
			Parser:      &H264Parser{},
		}
	default:
		return fmt.Errorf("unsupported video codec %q", c.vcodec)
	}
	switch c.acodec {
	case "OPUS":
		adeframer = &Deframer{
			SSRC:        c.assrc,
			PayloadType: c.apayload,
			ClockRate:   48000,
			Parser:      NullParser{Info: opus.NewCodecData(2)},
		}
	default:
		return fmt.Errorf("unsupported audio codec %q", c.acodec)
	}
	// setup RTP receiver
	ip := c.conn.RemoteAddr().(*net.TCPAddr).IP
	rch := make(chan []byte, 256)
	pktSrc := &rtpReader{
		ctx:        c.ctx,
		rtpPackets: rch,
		deframers:  []*Deframer{vdeframer /*, adeframer FIXME*/},
	}
	_ = adeframer
	hashKeys := c.hashKeys(ip)
	c.s.addReceiver(hashKeys, rch)
	remote := ip.String()
	log.Printf("[ftl] user %s started publishing to %s from %s", c.userID, c.channelName, remote)
	go func() {
		defer c.s.delReceiver(hashKeys, rch)
		if err := c.s.Publish(c.channelName, c.userID, remote, pktSrc); err != nil {
			log.Printf("[ftl] error: publishing %s from %s: %s", c.channelName, remote, err)
			c.cancel()
		} else {
			log.Printf("[ftl] publish of %s stopped", c.channelName)
		}
	}()

	listenPort := c.s.RTPAdvertisePort
	if listenPort == 0 {
		listenPort = c.s.RTPSocket.LocalAddr().(*net.UDPAddr).Port
	}
	_, err := fmt.Fprintf(c.tpc.W, "200 OK. Use UDP port %d\n", listenPort)
	return err
}

func (c *Conn) hashKeys(ip net.IP) []string {
	if ip4 := ip.To4(); ip4 != nil {
		ip = ip4
	}
	pkey := string(ip)
	keyb := make([]byte, len(ip)+4)
	copy(keyb[4:], ip)
	binary.BigEndian.PutUint32(keyb, c.vssrc)
	vkey := string(keyb)
	binary.BigEndian.PutUint32(keyb, c.assrc)
	akey := string(keyb)
	return []string{pkey, akey, vkey}
}
