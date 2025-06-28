package ssz

import (
	"encoding/binary"
	"unsafe"
)

var isSysLittleEndian bool

func init() {
	buf := [2]byte{}
	*(*uint16)(unsafe.Pointer(&buf[0])) = uint16(0xABCD)

	switch buf {
	case [2]byte{0xCD, 0xAB}:
		isSysLittleEndian = true
	case [2]byte{0xAB, 0xCD}:
		isSysLittleEndian = false
	default:
		panic("Could not determine native endianness.")
	}
}

func Uint64FromBytes(v []byte) uint64 {
	if isSysLittleEndian {
		return *(*uint64)(unsafe.Pointer(&v[0]))
	}
	return binary.LittleEndian.Uint64(v[:8])
}

func Uint32FromBytes(v []byte) uint32 {
	if isSysLittleEndian {
		return *(*uint32)(unsafe.Pointer(&v[0]))
	}
	return binary.LittleEndian.Uint32(v[:4])
}

func Uint16FromBytes(v []byte) uint16 {
	if isSysLittleEndian {
		return *(*uint16)(unsafe.Pointer(&v[0]))
	}
	return binary.LittleEndian.Uint16(v[:2])
}
