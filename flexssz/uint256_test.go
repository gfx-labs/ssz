package flexssz

import (
	"testing"
	
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUint256Pointer(t *testing.T) {
	t.Run("encode struct with uint256 pointer", func(t *testing.T) {
		type WithPointer struct {
			Value *uint256.Int `ssz:"uint256"`
		}
		
		// Test validation
		MustPrecacheStructSSZInfo(WithPointer{})
		
		// Test encoding
		val := uint256.NewInt(0x123456789ABCDEF0)
		s := WithPointer{
			Value: val,
		}
		
		encoded, err := EncodeStruct(s)
		require.NoError(t, err)
		assert.Len(t, encoded, 32) // uint256 is 32 bytes
		
		// Verify the encoding is correct
		expected := make([]byte, 32)
		expected[0] = 0xF0
		expected[1] = 0xDE
		expected[2] = 0xBC
		expected[3] = 0x9A
		expected[4] = 0x78
		expected[5] = 0x56
		expected[6] = 0x34
		expected[7] = 0x12
		assert.Equal(t, expected, encoded)
	})
	
	t.Run("encode struct with uint128 pointer", func(t *testing.T) {
		type WithPointer struct {
			Value *uint256.Int `ssz:"uint128"`
		}
		
		// Test validation
		MustPrecacheStructSSZInfo(WithPointer{})
		
		// Test encoding
		val := uint256.NewInt(0x123456789ABCDEF0)
		s := WithPointer{
			Value: val,
		}
		
		encoded, err := EncodeStruct(s)
		require.NoError(t, err)
		assert.Len(t, encoded, 16) // uint128 is 16 bytes
		
		// Verify the encoding is correct (only lower 16 bytes)
		expected := make([]byte, 16)
		expected[0] = 0xF0
		expected[1] = 0xDE
		expected[2] = 0xBC
		expected[3] = 0x9A
		expected[4] = 0x78
		expected[5] = 0x56
		expected[6] = 0x34
		expected[7] = 0x12
		assert.Equal(t, expected, encoded)
	})
	
	t.Run("nil pointer error", func(t *testing.T) {
		type WithPointer struct {
			Value *uint256.Int `ssz:"uint256"`
		}
		
		s := WithPointer{
			Value: nil,
		}
		
		_, err := EncodeStruct(s)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot encode nil pointer")
	})
	
	t.Run("mixed value and pointer types", func(t *testing.T) {
		type Mixed struct {
			DirectValue  uint256.Int  `ssz:"uint256"`
			PointerValue *uint256.Int `ssz:"uint256"`
			Uint128Ptr   *uint256.Int `ssz:"uint128"`
		}
		
		// Test validation
		MustPrecacheStructSSZInfo(Mixed{})
		
		// Test encoding
		val1 := uint256.NewInt(100)
		val2 := uint256.NewInt(200)
		val3 := uint256.NewInt(300)
		
		s := Mixed{
			DirectValue:  *val1,
			PointerValue: val2,
			Uint128Ptr:   val3,
		}
		
		encoded, err := EncodeStruct(s)
		require.NoError(t, err)
		assert.Len(t, encoded, 32+32+16) // uint256 + uint256 + uint128
	})
	
	t.Run("struct with multiple uint256 pointers", func(t *testing.T) {
		type WithMultiplePointers struct {
			First  *uint256.Int `ssz:"uint256"`
			Second *uint256.Int `ssz:"uint128"`
			Third  uint256.Int  `ssz:"uint256"`
		}
		
		// Test validation
		MustPrecacheStructSSZInfo(WithMultiplePointers{})
		
		// Test encoding
		val1 := uint256.NewInt(100)
		val2 := uint256.NewInt(200)
		val3 := uint256.NewInt(300)
		
		s := WithMultiplePointers{
			First:  val1,
			Second: val2,
			Third:  *val3,
		}
		
		encoded, err := EncodeStruct(s)
		require.NoError(t, err)
		assert.Len(t, encoded, 32+16+32) // uint256 + uint128 + uint256
	})
}