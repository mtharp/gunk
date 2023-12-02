package h264util

import "fmt"

type UnitType uint8

const (
	// Basic types
	TypeUnspecified UnitType = iota
	TypeNonIDR
	TypePartitionA
	TypePartitionB
	TypePartitionC
	TypeIDR
	TypeSupplementalEnhancement
	TypeSequenceParameter
	TypePictureParameter
	TypeAccessDelimiter
	TypeEndSequence
	TypeEndStream
	TypeFiller
	TypeSPSExtension
	TypePrefix
	TypeSubsetSPS
	TypeDepthParameter
	_
	_
	TypeAuxiliary
	TypeSliceExtension
	TypeDepthExtension
	_
	_
	// RTP framing
	TypeSTAPA
	TypeSTAPB
	TypeMTAP16
	TypeMTAP24
	TypeFragA
	TypeFragB
)

func NALType(nalu []byte) UnitType {
	if len(nalu) == 0 {
		return TypeUnspecified
	}
	return UnitType(nalu[0] & 0x1f)
}

func (t UnitType) String() string {
	switch t {
	case TypeNonIDR:
		return "Sli"
	case TypePartitionA:
		return "Pa"
	case TypePartitionB:
		return "Pb"
	case TypePartitionC:
		return "Pc"
	case TypeIDR:
		return "IDR"
	case TypeSupplementalEnhancement:
		return "SEI"
	case TypeSequenceParameter:
		return "SPS"
	case TypePictureParameter:
		return "PPS"
	case TypeAccessDelimiter:
		return "aud"
	case TypeEndSequence:
		return "EOQ"
	case TypeEndStream:
		return "EOS"
	case TypeFiller:
		return "fil"
	case TypeSPSExtension:
		return "SPE"
	case TypePrefix:
		return "Pfx"
	case TypeSubsetSPS:
		return "Sub"
	case TypeDepthParameter:
		return "DPS"
	case TypeAuxiliary:
		return "Aux"
	case TypeSliceExtension:
		return "Sle"
	case TypeDepthExtension:
		return "Dex"
	case TypeSTAPA:
		return "ST-A"
	case TypeSTAPB:
		return "ST-B"
	case TypeMTAP16:
		return "MT-16"
	case TypeMTAP24:
		return "MT-24"
	case TypeFragA:
		return "FU-A"
	case TypeFragB:
		return "FU-B"
	default:
		return fmt.Sprintf("%d", t)
	}
}
