package spectests

import (
	"fmt"
	"math/bits"
)

// BitList represents a variable-length bit array with SSZ encoding support
// The last byte contains a delimiter bit to indicate the actual length
type BitList []byte

// BitVector represents a fixed-length bit array
type BitVector []byte

// NewBitList creates a new BitList with the specified number of bits
func NewBitList(numBits uint64) BitList {
	if numBits == 0 {
		return BitList{0x01} // Empty bitlist with just the length bit
	}
	
	// Calculate required bytes: bits/8 + 1 for length bit
	numBytes := (numBits + 7) / 8
	
	// Create byte slice
	bl := make(BitList, numBytes)
	
	// Set the delimiter bit
	delimiterBitIndex := numBits % 8
	if delimiterBitIndex == 0 {
		// Need an extra byte for the delimiter
		bl = append(bl, 0x01)
	} else {
		// Set the delimiter bit in the last byte
		bl[numBytes-1] |= 1 << delimiterBitIndex
	}
	
	return bl
}

// NewBitVector creates a new BitVector with the specified number of bits
func NewBitVector(numBits uint64) BitVector {
	numBytes := (numBits + 7) / 8
	return make(BitVector, numBytes)
}

// Len returns the number of bits in the BitList (excluding the delimiter bit)
func (bl BitList) Len() uint64 {
	if len(bl) == 0 {
		return 0
	}
	
	// Find the delimiter bit in the last byte
	lastByte := bl[len(bl)-1]
	if lastByte == 0 {
		// Invalid bitlist - no delimiter bit
		return 0
	}
	
	// Find position of the most significant bit (delimiter)
	msb := bits.Len8(lastByte)
	
	// Calculate total bits: (bytes-1)*8 + position - 1
	return uint64((len(bl)-1)*8 + msb - 1)
}

// SetBit sets the bit at the given index to 1
func (bl BitList) SetBit(index uint64) error {
	bitLen := bl.Len()
	if index >= bitLen {
		return fmt.Errorf("index %d out of range for bitlist of length %d", index, bitLen)
	}
	
	byteIndex := index / 8
	bitIndex := index % 8
	bl[byteIndex] |= 1 << bitIndex
	return nil
}

// GetBit returns the value of the bit at the given index
func (bl BitList) GetBit(index uint64) (bool, error) {
	bitLen := bl.Len()
	if index >= bitLen {
		return false, fmt.Errorf("index %d out of range for bitlist of length %d", index, bitLen)
	}
	
	byteIndex := index / 8
	bitIndex := index % 8
	return (bl[byteIndex] & (1 << bitIndex)) != 0, nil
}

// SetBit sets the bit at the given index to 1
func (bv BitVector) SetBit(index uint64) error {
	if index >= uint64(len(bv)*8) {
		return fmt.Errorf("index %d out of range for bitvector of length %d", index, len(bv)*8)
	}
	
	byteIndex := index / 8
	bitIndex := index % 8
	bv[byteIndex] |= 1 << bitIndex
	return nil
}

// GetBit returns the value of the bit at the given index
func (bv BitVector) GetBit(index uint64) (bool, error) {
	if index >= uint64(len(bv)*8) {
		return false, fmt.Errorf("index %d out of range for bitvector of length %d", index, len(bv)*8)
	}
	
	byteIndex := index / 8
	bitIndex := index % 8
	return (bv[byteIndex] & (1 << bitIndex)) != 0, nil
}

// Len returns the number of bits in the BitVector
func (bv BitVector) Len() uint64 {
	return uint64(len(bv) * 8)
}

// ValidateBitList validates that a BitList has proper format
func ValidateBitList(bl BitList, maxBits uint64) error {
	if len(bl) == 0 {
		return fmt.Errorf("bitlist empty, it does not have length bit")
	}
	
	// Maximum possible bytes in a bitlist with provided bitlimit
	maxBytes := (maxBits >> 3) + 1
	if uint64(len(bl)) > maxBytes {
		return fmt.Errorf("bitlist exceeds maximum size: got %d bytes, max %d bytes", len(bl), maxBytes)
	}
	
	// Check for delimiter bit
	lastByte := bl[len(bl)-1]
	if lastByte == 0 {
		return fmt.Errorf("bitlist missing delimiter bit")
	}
	
	// Check actual bit length doesn't exceed limit
	bitLen := bl.Len()
	if bitLen > maxBits {
		return fmt.Errorf("bitlist has %d bits, exceeds limit of %d", bitLen, maxBits)
	}
	
	return nil
}

// Bytes returns the underlying byte representation
func (bl BitList) Bytes() []byte {
	return []byte(bl)
}

// Bytes returns the underlying byte representation
func (bv BitVector) Bytes() []byte {
	return []byte(bv)
}