package solidssz

import (
	"fmt"
)

// Bitlist represents an SSZ bitlist with a maximum capacity.
// The bitlist is encoded with a sentinel bit to indicate the actual length.
type Bitlist struct {
	data  []byte
	limit uint64 // Maximum number of bits
}

// NewBitlist creates a new bitlist with the specified limit.
func NewBitlist(limit uint64) *Bitlist {
	return &Bitlist{
		data:  []byte{0x01}, // Start with just the sentinel bit at position 0
		limit: limit,
	}
}

// NewBitlistFromSSZ creates a bitlist from SSZ-encoded bytes.
// The bytes should include the sentinel bit.
func NewBitlistFromSSZ(data []byte, limit uint64) (*Bitlist, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("bitlist data cannot be empty")
	}

	maxBytes := (limit + 7) / 8 + 1 // +1 for potential sentinel bit overflow
	if uint64(len(data)) > maxBytes {
		return nil, fmt.Errorf("bitlist data exceeds maximum size: got %d bytes, max %d", len(data), maxBytes)
	}

	// Make a copy of the data
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)

	return &Bitlist{
		data:  dataCopy,
		limit: limit,
	}, nil
}

// Len returns the number of bits in the bitlist (not including the sentinel).
func (b *Bitlist) Len() int {
	if len(b.data) == 0 {
		return 0
	}

	// Find the highest set bit (the sentinel)
	for i := len(b.data) - 1; i >= 0; i-- {
		if b.data[i] != 0 {
			// Find the position of the highest bit in this byte
			for j := 7; j >= 0; j-- {
				if b.data[i]&(1<<uint(j)) != 0 {
					// This is the sentinel bit
					return i*8 + j
				}
			}
		}
	}
	return 0
}

// Get returns the bit at the specified index.
func (b *Bitlist) Get(index int) (bool, error) {
	if index < 0 || index >= b.Len() {
		return false, fmt.Errorf("index %d out of bounds [0, %d)", index, b.Len())
	}

	byteIndex := index / 8
	bitIndex := uint(index % 8)
	return b.data[byteIndex]&(1<<bitIndex) != 0, nil
}

// Set sets the bit at the specified index.
func (b *Bitlist) Set(index int, value bool) error {
	if index < 0 || uint64(index) >= b.limit {
		return fmt.Errorf("index %d out of bounds [0, %d)", index, b.limit)
	}

	// Ensure we have enough bytes
	requiredBytes := (index / 8) + 1
	currentLen := b.Len()

	// If we're extending the bitlist
	if index >= currentLen {
		// We need to move the sentinel bit
		newDataLen := requiredBytes
		if newDataLen > len(b.data) {
			// Grow the data slice
			newData := make([]byte, newDataLen)
			copy(newData, b.data)
			b.data = newData
		}

		// Clear the old sentinel bit if it exists
		if currentLen > 0 {
			oldByteIndex := currentLen / 8
			oldBitIndex := uint(currentLen % 8)
			if oldByteIndex < len(b.data) {
				b.data[oldByteIndex] &^= (1 << oldBitIndex)
			}
		}

		// Set the new sentinel bit
		newSentinelIndex := index + 1
		newSentinelByte := newSentinelIndex / 8
		newSentinelBit := uint(newSentinelIndex % 8)

		// Ensure we have space for the sentinel
		if newSentinelByte >= len(b.data) {
			newData := make([]byte, newSentinelByte+1)
			copy(newData, b.data)
			b.data = newData
		}

		b.data[newSentinelByte] |= (1 << newSentinelBit)
	}

	// Set or clear the bit
	byteIndex := index / 8
	bitIndex := uint(index % 8)
	if value {
		b.data[byteIndex] |= (1 << bitIndex)
	} else {
		b.data[byteIndex] &^= (1 << bitIndex)
	}

	return nil
}

// Append adds a bit to the end of the bitlist.
func (b *Bitlist) Append(value bool) error {
	currentLen := b.Len()
	if uint64(currentLen) >= b.limit {
		return fmt.Errorf("bitlist is at maximum capacity (%d bits)", b.limit)
	}

	return b.Set(currentLen, value)
}

// SSZBytes returns the SSZ encoding of the bitlist (includes sentinel bit).
func (b *Bitlist) SSZBytes() []byte {
	// Return a copy to prevent external modification
	result := make([]byte, len(b.data))
	copy(result, b.data)
	return result
}

// SizeSSZ returns the size of the SSZ encoding in bytes.
func (b *Bitlist) SizeSSZ() int {
	return len(b.data)
}

// MaxSize returns the maximum size in bytes when serialized.
func (b *Bitlist) MaxSize() uint64 {
	// Maximum bytes needed for limit bits plus sentinel
	return (b.limit + 7) / 8 + 1
}

// Clear removes all bits from the bitlist.
func (b *Bitlist) Clear() {
	b.data = []byte{0x01} // Reset to just the sentinel bit at position 0
}

// String returns a string representation of the bitlist.
func (b *Bitlist) String() string {
	length := b.Len()
	if length == 0 {
		return "[]"
	}

	result := make([]byte, length)
	for i := 0; i < length; i++ {
		bit, _ := b.Get(i)
		if bit {
			result[i] = '1'
		} else {
			result[i] = '0'
		}
	}
	return fmt.Sprintf("[%s]", string(result))
}