package flexssz

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBitListEncoding(t *testing.T) {
	tests := []struct {
		name     string
		bits     []byte
		maxBits  int
		expected []byte
	}{
		{
			name:     "empty bitlist",
			bits:     []byte{},
			maxBits:  100,
			expected: []byte{0x01}, // Just delimiter bit
		},
		{
			name:     "single bit set",
			bits:     []byte{0x01},
			maxBits:  100,
			expected: []byte{0x03}, // 0000 0011 - bit 0 set, delimiter at bit 1
		},
		{
			name:     "multiple bits",
			bits:     []byte{0xFF}, // 11111111
			maxBits:  100,
			expected: []byte{0xFF, 0x01}, // All bits set, delimiter in next byte
		},
		{
			name:     "trailing zeros removed",
			bits:     []byte{0x0F, 0x00, 0x00}, // 00001111 00000000 00000000
			maxBits:  100,
			expected: []byte{0x1F}, // 00011111 - bits 0-3 set, delimiter at bit 4
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := EncodeBitList(tt.bits, tt.maxBits)
			require.NoError(t, err)
			require.Equal(t, tt.expected, encoded, "encoded bytes should match")

			// Test round-trip
			decoded, numBits, err := DecodeBitList(encoded, tt.maxBits)
			require.NoError(t, err)
			
			// Compare only the relevant bits
			if len(tt.bits) > 0 {
				relevantBytes := (numBits + 7) / 8
				require.True(t, bytes.Equal(tt.bits[:relevantBytes], decoded[:relevantBytes]), 
					"decoded bits should match original")
			}
		})
	}
}

func TestBitListMaxSize(t *testing.T) {
	// Test exceeding max size
	bits := make([]byte, 10) // 80 bits
	_, err := EncodeBitList(bits, 50) // Max 50 bits
	require.Error(t, err)
	require.Contains(t, err.Error(), "exceeds maximum")
}

func TestBitVectorEncoding(t *testing.T) {
	tests := []struct {
		name     string
		bits     []byte
		size     int
		expected []byte
	}{
		{
			name:     "8 bit vector",
			bits:     []byte{0xFF},
			size:     8,
			expected: []byte{0xFF},
		},
		{
			name:     "16 bit vector",
			bits:     []byte{0xFF, 0x00},
			size:     16,
			expected: []byte{0xFF, 0x00},
		},
		{
			name:     "5 bit vector - extra bits cleared",
			bits:     []byte{0xFF},
			size:     5,
			expected: []byte{0x1F}, // Only lower 5 bits kept
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := EncodeBitVector(tt.bits, tt.size)
			require.NoError(t, err)
			require.Equal(t, tt.expected, encoded)

			// Test round-trip
			decoded, err := DecodeBitVector(encoded, tt.size)
			require.NoError(t, err)
			require.Equal(t, tt.expected, decoded)
		})
	}
}

func TestBitVectorValidation(t *testing.T) {
	// Wrong size
	_, err := EncodeBitVector([]byte{0xFF, 0xFF}, 8) // 2 bytes for 8 bits
	require.Error(t, err)
	require.Contains(t, err.Error(), "requires exactly 1 bytes")

	// Extra bits set
	_, err = DecodeBitVector([]byte{0xFF}, 5) // Upper 3 bits should be 0
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid bits set")
}

func TestBitListHelpers(t *testing.T) {
	bits := NewBitList(20) // 20 bits = 3 bytes
	require.Len(t, bits, 3)

	// Set some bits
	err := SetBit(bits, 0)
	require.NoError(t, err)
	err = SetBit(bits, 5)
	require.NoError(t, err)
	err = SetBit(bits, 19)
	require.NoError(t, err)

	// Check bits
	val, err := GetBit(bits, 0)
	require.NoError(t, err)
	require.True(t, val)

	val, err = GetBit(bits, 5)
	require.NoError(t, err)
	require.True(t, val)

	val, err = GetBit(bits, 19)
	require.NoError(t, err)
	require.True(t, val)

	val, err = GetBit(bits, 1)
	require.NoError(t, err)
	require.False(t, val)

	// Out of range
	err = SetBit(bits, 24)
	require.Error(t, err)
}

