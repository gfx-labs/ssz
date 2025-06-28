package ssz

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUint64FromBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected uint64
	}{
		{
			name:     "zero value",
			input:    []byte{0, 0, 0, 0, 0, 0, 0, 0},
			expected: 0,
		},
		{
			name:     "max value",
			input:    []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
			expected: 0xFFFFFFFFFFFFFFFF,
		},
		{
			name:     "little endian byte order",
			input:    []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			expected: 0x0807060504030201,
		},
		{
			name:     "single byte set",
			input:    []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			expected: 1,
		},
		{
			name:     "high byte set",
			input:    []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
			expected: 0x0100000000000000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Uint64FromBytes(tt.input)
			assert.Equal(t, tt.expected, result, "Uint64FromBytes(%v)", tt.input)
			
			// Verify against standard library
			stdResult := binary.LittleEndian.Uint64(tt.input)
			assert.Equal(t, stdResult, result, "Should match standard library result")
		})
	}
}

func TestUint32FromBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected uint32
	}{
		{
			name:     "zero value",
			input:    []byte{0, 0, 0, 0},
			expected: 0,
		},
		{
			name:     "max value",
			input:    []byte{0xFF, 0xFF, 0xFF, 0xFF},
			expected: 0xFFFFFFFF,
		},
		{
			name:     "little endian byte order",
			input:    []byte{0x01, 0x02, 0x03, 0x04},
			expected: 0x04030201,
		},
		{
			name:     "single byte set",
			input:    []byte{0x01, 0x00, 0x00, 0x00},
			expected: 1,
		},
		{
			name:     "high byte set",
			input:    []byte{0x00, 0x00, 0x00, 0x01},
			expected: 0x01000000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Uint32FromBytes(tt.input)
			assert.Equal(t, tt.expected, result, "Uint32FromBytes(%v)", tt.input)
			
			// Verify against standard library
			stdResult := binary.LittleEndian.Uint32(tt.input)
			assert.Equal(t, stdResult, result, "Should match standard library result")
		})
	}
}

func TestUint16FromBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected uint16
	}{
		{
			name:     "zero value",
			input:    []byte{0, 0},
			expected: 0,
		},
		{
			name:     "max value",
			input:    []byte{0xFF, 0xFF},
			expected: 0xFFFF,
		},
		{
			name:     "little endian byte order",
			input:    []byte{0x01, 0x02},
			expected: 0x0201,
		},
		{
			name:     "single byte set",
			input:    []byte{0x01, 0x00},
			expected: 1,
		},
		{
			name:     "high byte set",
			input:    []byte{0x00, 0x01},
			expected: 0x0100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Uint16FromBytes(tt.input)
			assert.Equal(t, tt.expected, result, "Uint16FromBytes(%v)", tt.input)
			
			// Verify against standard library
			stdResult := binary.LittleEndian.Uint16(tt.input)
			assert.Equal(t, stdResult, result, "Should match standard library result")
		})
	}
}

func TestUint64FromBytesWithExtraBytes(t *testing.T) {
	// Test that function works correctly when given more than 8 bytes
	input := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A}
	expected := uint64(0x0807060504030201)
	
	require.GreaterOrEqual(t, len(input), 8, "Input must have at least 8 bytes")
	result := Uint64FromBytes(input)
	assert.Equal(t, expected, result, "Uint64FromBytes should handle extra bytes correctly")
}

func TestUint32FromBytesWithExtraBytes(t *testing.T) {
	// Test that function works correctly when given more than 4 bytes
	input := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06}
	expected := uint32(0x04030201)
	
	result := Uint32FromBytes(input)
	assert.Equal(t, expected, result, "Uint32FromBytes should handle extra bytes correctly")
}

func TestUint16FromBytesWithExtraBytes(t *testing.T) {
	// Test that function works correctly when given more than 2 bytes
	input := []byte{0x01, 0x02, 0x03, 0x04}
	expected := uint16(0x0201)
	
	result := Uint16FromBytes(input)
	assert.Equal(t, expected, result, "Uint16FromBytes should handle extra bytes correctly")
}

func BenchmarkUint64FromBytes(b *testing.B) {
	data := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Uint64FromBytes(data)
	}
}

func BenchmarkUint64FromBytesStdLib(b *testing.B) {
	data := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = binary.LittleEndian.Uint64(data)
	}
}

func BenchmarkUint32FromBytes(b *testing.B) {
	data := []byte{0x01, 0x02, 0x03, 0x04}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Uint32FromBytes(data)
	}
}

func BenchmarkUint32FromBytesStdLib(b *testing.B) {
	data := []byte{0x01, 0x02, 0x03, 0x04}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = binary.LittleEndian.Uint32(data)
	}
}

func BenchmarkUint16FromBytes(b *testing.B) {
	data := []byte{0x01, 0x02}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Uint16FromBytes(data)
	}
}

func BenchmarkUint16FromBytesStdLib(b *testing.B) {
	data := []byte{0x01, 0x02}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = binary.LittleEndian.Uint16(data)
	}
}