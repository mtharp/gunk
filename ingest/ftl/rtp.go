package ftl

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"time"

	"github.com/kr/pretty"
	"github.com/nareix/joy4/av"
	"github.com/pion/rtp"
)

func (s *Server) serveRTP() {
	for {
		d := make([]byte, 1500)
		n, addr, err := s.RTPSocket.ReadFrom(d)
		if err != nil {
			log.Println("error: receiving from UDP socket:", err)
			time.Sleep(time.Second)
			continue
		}
		d = d[:n]
		if len(d) < 12 {
			continue
		}
		// identify sender by IP and SSRC
		ip := addr.(*net.UDPAddr).IP
		if ip4 := ip.To4(); ip4 != nil {
			ip = ip4
		}
		if d[0] == 0x81 && d[1] == 0xfa {
			// ping packet
			key := string(ip)
			s.mu.Lock()
			ok := s.receivers[key] != nil
			s.mu.Unlock()
			if ok {
				s.RTPSocket.WriteTo(d, addr)
			}
			continue
		} else if d[1] == 0xc8 {
			// RTCP sender report
			continue
		}
		ssrc := d[8:12]
		keyb := make([]byte, len(ip)+4)
		copy(keyb, ssrc)
		copy(keyb[4:], ip)
		key := string(keyb)
		s.mu.Lock()
		rcv := s.receivers[key]
		s.mu.Unlock()
		if rcv == nil {
			continue
		}
		select {
		case rcv <- d:
		default:
			log.Printf("[ftl] %s overflow in UDP handler", addr)
		}
	}
}

func (s *Server) addReceiver(keys []string, rch chan<- []byte) {
	s.mu.Lock()
	if s.receivers == nil {
		s.receivers = make(map[string]chan<- []byte)
	}
	for _, k := range keys {
		s.receivers[k] = rch
	}
	s.mu.Unlock()
}

func (s *Server) delReceiver(keys []string, rch chan<- []byte) {
	s.mu.Lock()
	for _, k := range keys {
		if r := s.receivers[k]; r == rch {
			delete(s.receivers, k)
		}
	}
	s.mu.Unlock()
}

type rtpReader struct {
	ctx        context.Context
	rtpPackets <-chan []byte
	deframers  []*Deframer
	streams    []av.CodecData
	saved      []av.Packet
}

func (r *rtpReader) readPacket(ctx context.Context) (av.Packet, error) {
	if len(r.saved) != 0 {
		pkt := r.saved[0]
		r.saved = r.saved[1:]
		if len(r.saved) == 0 {
			r.saved = nil
		}
		return pkt, nil
	}
	for {
		select {
		case <-ctx.Done():
			return av.Packet{}, io.EOF
		case d, ok := <-r.rtpPackets:
			if !ok {
				return av.Packet{}, io.EOF
			}
			pkt := new(rtp.Packet)
			if err := pkt.Unmarshal(d); err != nil {
				return av.Packet{}, err
			}
			for i, def := range r.deframers {
				if pkt.SSRC != def.SSRC || pkt.PayloadType != def.PayloadType {
					continue
				}
				packets, err := def.Deframe(pkt)
				if err != nil {
					return av.Packet{}, err
				}
				for j := range packets {
					packets[j].Idx = int8(i)
				}
				if len(packets) == 1 {
					return packets[0], nil
				} else if len(packets) > 1 {
					r.saved = packets[1:]
					return packets[0], nil
				}
				// only fragments, no full packet produced
				break
			}
			// no match, try again
		}
	}
}

func (r *rtpReader) Streams() ([]av.CodecData, error) {
	if r.streams != nil {
		return r.streams, nil
	}
	log.Printf("getting streams")
	ctx, cancel := context.WithTimeout(r.ctx, 10*time.Second)
	defer cancel()
	streams := make([]av.CodecData, len(r.deframers))
	for ctx.Err() == nil {
		pkt, err := r.readPacket(ctx)
		if err != nil {
			return nil, err
		}
		i := int(pkt.Idx)
		def := r.deframers[i]
		if cd, err := def.Parser.CodecData(); err != nil {
			return nil, err
		} else if cd != nil {
			if streams[i] == nil {
				pretty.Println("stream", i, cd)
			}
			streams[i] = cd
		}
		ready := true
		for _, cd := range streams {
			if cd == nil {
				ready = false
			}
		}
		if ready {
			r.streams = streams
			return streams, nil
		}
	}
	return nil, errors.New("timed out waiting for codec data")
}

func (r *rtpReader) ReadPacket() (av.Packet, error) {
	if r.streams == nil {
		_, err := r.Streams()
		if err != nil {
			return av.Packet{}, err
		}
	}
	return r.readPacket(r.ctx)
}
