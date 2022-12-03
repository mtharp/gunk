package rist

import (
	"encoding/binary"
	"errors"
	"fmt"
)

type (
	ristVersion uint8
	subtype     uint16
)

const (
	etypeLegacyKeepAlive = 0x88b5
	etypeLegacyPacket    = 0x88b6
	etypeVSF             = 0xcce0

	subtypePacket      subtype = 0x0000
	subtypeKeepAlive   subtype = 0x8000
	subtypeFutureNonce subtype = 0x8001

	ristVersion2020 ristVersion = 0
	ristVersion2021 ristVersion = 1
	ristVersion2022 ristVersion = 2
)

type encapsulatedPacket struct {
	Payload      []byte
	RISTVersion  uint8
	Sequence     uint32
	Nonce        uint32
	Source, Dest uint16
	Version      ristVersion
	Subtype      subtype
	LongPSK      bool
}

func (p *encapsulatedPacket) Parse(d []byte) error {
	if len(d) < 4 {
		return errors.New("short packet")
	}
	flags, verbits := d[0], d[1]
	p.Version = ristVersion((verbits >> 3) & 7)
	if p.Version > 4 {
		return fmt.Errorf("unrecognized RIST version %d", p.Version)
	}
	etype := binary.BigEndian.Uint16(d[2:])
	d = d[4:]
	if flags&0x80 != 0 {
		// checksum present (ignored)
		if len(d) < 4 {
			return errors.New("short packet in GRE checksum")
		}
		d = d[4:]
	}
	if flags&0x20 != 0 {
		// key present
		if len(d) < 4 {
			return errors.New("short packet in GRE key")
		}
		p.Nonce = binary.BigEndian.Uint32(d)
		d = d[4:]
	}
	if flags&0x10 != 0 {
		// sequence present
		if len(d) < 4 {
			return errors.New("short packet in GRE sequence")
		}
		p.Sequence = binary.BigEndian.Uint32(d)
		d = d[4:]
	}
	switch etype {
	case etypeLegacyKeepAlive:
		p.Subtype = subtypeKeepAlive
	case etypeLegacyPacket:
		p.Subtype = subtypePacket
	case etypeVSF:
		// VSF ethertype
		if len(d) < 4 {
			return errors.New("short packet in VSF header")
		}
		major := binary.BigEndian.Uint16(d)
		if major != 0 {
			return fmt.Errorf("unsupported VSF protocol type 0x%04x", major)
		}
		p.Subtype = subtype(binary.BigEndian.Uint16(d[2:]))
		d = d[4:]
	default:
		return fmt.Errorf("unsupported ethertype 0x%04x", etype)
	}
	switch p.Subtype {
	case subtypePacket:
		if len(d) < 4 {
			return errors.New("short packet in GRE payload")
		}
		p.Source = binary.BigEndian.Uint16(d)
		p.Dest = binary.BigEndian.Uint16(d[2:])
		d = d[4:]
	case subtypeKeepAlive, subtypeFutureNonce:
		// TODO
	default:
		return fmt.Errorf("unsupported VSF subtype 0x%04x", p.Subtype)
	}
	p.Payload = d
	return nil
}
