package flexssz

import (
	"fmt"
)

// EncodeBitList encodes a bitlist to SSZ format.
// A bitlist is a []byte where the last byte has a delimiter bit set to indicate the end.
// The bits are packed into bytes in little-endian order (bit 0 is the LSB of byte 0).
func EncodeBitList(bits []byte, maxBits int) ([]byte, error) {
	if len(bits) == 0 {
		// Empty bitlist is encoded as a single byte with delimiter bit
		return []byte{0x01}, nil
	}

	// Check maximum size
	if maxBits > 0 && len(bits)*8 > maxBits {
		return nil, fmt.Errorf("bitlist length %d exceeds maximum %d bits", len(bits)*8, maxBits)
	}

	// Copy the bits
	result := make([]byte, len(bits))
	copy(result, bits)

	// Find the last byte with actual data (non-zero)
	lastNonZero := len(result) - 1
	for lastNonZero >= 0 && result[lastNonZero] == 0 {
		lastNonZero--
	}

	// If all bytes are zero, we still need at least one byte
	if lastNonZero < 0 {
		return []byte{0x01}, nil
	}

	// Trim to the last non-zero byte + 1
	result = result[:lastNonZero+1]

	// Set the delimiter bit in the last byte
	// Find the highest set bit in the last byte
	lastByte := result[len(result)-1]
	delimiterBit := uint8(0x80)
	for delimiterBit > lastByte {
		delimiterBit >>= 1
	}
	// Move delimiter bit one position higher
	delimiterBit <<= 1
	if delimiterBit == 0 {
		// Need to add a new byte for the delimiter
		result = append(result, 0x01)
	} else {
		result[len(result)-1] |= delimiterBit
	}

	return result, nil
}

// DecodeBitList decodes a bitlist from SSZ format.
// Returns the bitlist as []byte and the number of bits.
func DecodeBitList(data []byte, maxBits int) ([]byte, int, error) {
	if len(data) == 0 {
		return nil, 0, fmt.Errorf("empty data for bitlist")
	}

	// Special case: single byte with only delimiter bit means empty bitlist
	if len(data) == 1 && data[0] == 0x01 {
		return []byte{}, 0, nil
	}

	// Find the delimiter bit in the last byte
	lastByte := data[len(data)-1]
	if lastByte == 0 {
		return nil, 0, fmt.Errorf("bitlist missing delimiter bit")
	}

	// Find the position of the delimiter bit
	delimiterBit := uint8(0x80)
	delimiterPos := 7
	for (lastByte & delimiterBit) == 0 {
		delimiterBit >>= 1
		delimiterPos--
	}

	// Calculate the number of bits
	numBits := (len(data)-1)*8 + delimiterPos

	// Check maximum size
	if maxBits > 0 && numBits > maxBits {
		return nil, 0, fmt.Errorf("bitlist has %d bits, exceeds maximum %d", numBits, maxBits)
	}

	// Create result without delimiter bit
	result := make([]byte, len(data))
	copy(result, data)
	
	// Clear the delimiter bit
	result[len(result)-1] &= ^delimiterBit

	// Trim trailing zero bytes that were just holding the delimiter
	for len(result) > 0 && result[len(result)-1] == 0 {
		result = result[:len(result)-1]
	}

	return result, numBits, nil
}

// EncodeBitVector encodes a bitvector to SSZ format.
// A bitvector is a fixed-size bit array without a delimiter bit.
func EncodeBitVector(bits []byte, size int) ([]byte, error) {
	expectedBytes := (size + 7) / 8
	
	if len(bits) != expectedBytes {
		return nil, fmt.Errorf("bitvector requires exactly %d bytes for %d bits, got %d bytes", expectedBytes, size, len(bits))
	}

	// For bitvector, we just return the bytes as-is
	result := make([]byte, len(bits))
	copy(result, bits)
	
	// Clear any extra bits in the last byte
	extraBits := size % 8
	if extraBits > 0 {
		mask := byte((1 << extraBits) - 1)
		result[len(result)-1] &= mask
	}

	return result, nil
}

// DecodeBitVector decodes a bitvector from SSZ format.
func DecodeBitVector(data []byte, size int) ([]byte, error) {
	expectedBytes := (size + 7) / 8
	
	if len(data) != expectedBytes {
		return nil, fmt.Errorf("bitvector requires exactly %d bytes for %d bits, got %d bytes", expectedBytes, size, len(data))
	}

	// Copy the data
	result := make([]byte, len(data))
	copy(result, data)

	// Verify no extra bits are set in the last byte
	extraBits := size % 8
	if extraBits > 0 {
		mask := byte((1 << extraBits) - 1)
		if (result[len(result)-1] & ^mask) != 0 {
			return nil, fmt.Errorf("bitvector has invalid bits set beyond size %d", size)
		}
	}

	return result, nil
}

// BitListHelpers - Helper functions for working with bitlists as []byte

// SetBit sets the bit at the given index in a []byte bitlist
func SetBit(bits []byte, index int) error {
	byteIndex := index / 8
	if byteIndex >= len(bits) {
		return fmt.Errorf("bit index %d out of range for bitlist of %d bytes", index, len(bits))
	}
	bitIndex := index % 8
	bits[byteIndex] |= 1 << bitIndex
	return nil
}

// GetBit gets the bit at the given index in a []byte bitlist
func GetBit(bits []byte, index int) (bool, error) {
	byteIndex := index / 8
	if byteIndex >= len(bits) {
		return false, fmt.Errorf("bit index %d out of range for bitlist of %d bytes", index, len(bits))
	}
	bitIndex := index % 8
	return (bits[byteIndex] & (1 << bitIndex)) != 0, nil
}

// NewBitList creates a new bitlist with the given number of bits
func NewBitList(numBits int) []byte {
	numBytes := (numBits + 7) / 8
	return make([]byte, numBytes)
}

// NewBitVector creates a new bitvector with the given number of bits
func NewBitVector(numBits int) []byte {
	numBytes := (numBits + 7) / 8
	return make([]byte, numBytes)
}