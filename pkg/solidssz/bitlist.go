package solidssz

import (
	"fmt"
)

// Bitlist represents an SSZ bitlist as raw bytes with a sentinel bit.
// The bitlist is encoded with a sentinel bit to indicate the actual length.
// This is a zero-copy type that directly wraps SSZ-encoded bytes.
type Bitlist []byte

// NewBitlist creates a new empty bitlist.
func NewBitlist() Bitlist {
	return Bitlist{0x01} // Start with just the sentinel bit at position 0
}

// NewBitlistFromSSZ creates a bitlist from SSZ-encoded bytes (zero-copy).
// The bytes should include the sentinel bit.
func NewBitlistFromSSZ(data []byte) Bitlist {
	return Bitlist(data)
}

// Len returns the number of bits in the bitlist (not including the sentinel).
func (b Bitlist) Len() int {
	if len(b) == 0 {
		return 0
	}

	// Find the highest set bit (the sentinel)
	for i := len(b) - 1; i >= 0; i-- {
		if b[i] != 0 {
			// Find the position of the highest bit in this byte
			for j := 7; j >= 0; j-- {
				if b[i]&(1<<uint(j)) != 0 {
					// This is the sentinel bit
					return i*8 + j
				}
			}
		}
	}
	return 0
}

// Get returns the bit at the specified index.
func (b Bitlist) Get(index int) (bool, error) {
	if index < 0 || index >= b.Len() {
		return false, fmt.Errorf("index %d out of bounds [0, %d)", index, b.Len())
	}

	byteIndex := index / 8
	bitIndex := uint(index % 8)
	return b[byteIndex]&(1<<bitIndex) != 0, nil
}

// Set sets the bit at the specified index.
func (b *Bitlist) Set(index int, value bool) error {
	if index < 0 {
		return fmt.Errorf("index %d out of bounds", index)
	}

	// Ensure we have enough bytes
	requiredBytes := (index / 8) + 1
	currentLen := b.Len()

	// If we're extending the bitlist
	if index >= currentLen {
		// We need to move the sentinel bit
		newDataLen := requiredBytes
		if newDataLen > len(*b) {
			// Grow the data slice
			newData := make([]byte, newDataLen)
			copy(newData, *b)
			*b = newData
		}

		// Clear the old sentinel bit if it exists
		if currentLen > 0 {
			oldByteIndex := currentLen / 8
			oldBitIndex := uint(currentLen % 8)
			if oldByteIndex < len(*b) {
				(*b)[oldByteIndex] &^= (1 << oldBitIndex)
			}
		}

		// Set the new sentinel bit
		newSentinelIndex := index + 1
		newSentinelByte := newSentinelIndex / 8
		newSentinelBit := uint(newSentinelIndex % 8)

		// Ensure we have space for the sentinel
		if newSentinelByte >= len(*b) {
			newData := make([]byte, newSentinelByte+1)
			copy(newData, *b)
			*b = newData
		}

		(*b)[newSentinelByte] |= (1 << newSentinelBit)
	}

	// Set or clear the bit
	byteIndex := index / 8
	bitIndex := uint(index % 8)
	if value {
		(*b)[byteIndex] |= (1 << bitIndex)
	} else {
		(*b)[byteIndex] &^= (1 << bitIndex)
	}

	return nil
}

// Append adds a bit to the end of the bitlist.
func (b *Bitlist) Append(value bool) error {
	currentLen := b.Len()
	return b.Set(currentLen, value)
}

// SSZBytes returns the SSZ encoding of the bitlist (includes sentinel bit).
func (b Bitlist) SSZBytes() []byte {
	return []byte(b)
}

// SizeSSZ returns the size of the SSZ encoding in bytes.
func (b Bitlist) SizeSSZ() int {
	return len(b)
}

// Clear removes all bits from the bitlist.
func (b *Bitlist) Clear() {
	*b = Bitlist{0x01} // Reset to just the sentinel bit at position 0
}

// String returns a string representation of the bitlist.
func (b Bitlist) String() string {
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
