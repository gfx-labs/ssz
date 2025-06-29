package flexssz

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRoundTrip verifies that we can encode and decode various struct types
func TestRoundTrip(t *testing.T) {
	t.Run("all types", func(t *testing.T) {
		type AllTypes struct {
			// Basic types
			U8     uint8        `ssz:"uint8"`
			U16    uint16       `ssz:"uint16"`
			U32    uint32       `ssz:"uint32"`
			U64    uint64       `ssz:"uint64"`
			Bool   bool         `ssz:"bool"`
			U128   uint256.Int  `ssz:"uint128"`
			U256   uint256.Int  `ssz:"uint256"`
			U256P  *uint256.Int `ssz:"uint256"`
			
			// Fixed arrays
			Hash   [32]byte     `ssz:"vector"`
			Nums   [4]uint32    `ssz:"vector"`
			
			// Variable types
			Str    string       `ssz:"string"`
			Bytes  []byte       `ssz:"list" ssz-max:"100"`
			Slice  []uint64     `ssz-max:"50"`
			
			// Nested struct
			Inner  struct {
				A uint32 `ssz:"uint32"`
				B string `ssz:"string"`
				C []byte `ssz:"list" ssz-max:"64"`
			}
		}
		
		// Create test data
		u256Val := uint256.NewInt(999999)
		original := AllTypes{
			U8:     255,
			U16:    65535,
			U32:    4294967295,
			U64:    18446744073709551615,
			Bool:   true,
			U256P:  u256Val,
			Str:    "hello world",
			Bytes:  []byte{1, 2, 3, 4, 5},
			Slice:  []uint64{100, 200, 300, 400, 500},
			Inner: struct {
				A uint32 `ssz:"uint32"`
				B string `ssz:"string"`
				C []byte `ssz:"list" ssz-max:"64"`
			}{
				A: 12345,
				B: "inner string",
				C: []byte("inner bytes"),
			},
		}
		
		// Set uint128/256 values
		original.U128.SetUint64(0xFFFFFFFFFFFFFFFF)
		original.U256.SetFromHex("0xDEADBEEFCAFEBABEDEADBEEFCAFEBABEDEADBEEFCAFEBABEDEADBEEFCAFEBABE")
		
		// Fill arrays
		for i := range original.Hash {
			original.Hash[i] = byte(i)
		}
		for i := range original.Nums {
			original.Nums[i] = uint32(i * 1000)
		}
		
		// Encode
		encoded, err := Marshal(original)
		require.NoError(t, err)
		
		// Decode
		var decoded AllTypes
		err = Unmarshal(encoded, &decoded)
		require.NoError(t, err)
		
		// Compare
		assert.Equal(t, original.U8, decoded.U8)
		assert.Equal(t, original.U16, decoded.U16)
		assert.Equal(t, original.U32, decoded.U32)
		assert.Equal(t, original.U64, decoded.U64)
		assert.Equal(t, original.Bool, decoded.Bool)
		assert.Equal(t, original.U128.String(), decoded.U128.String())
		assert.Equal(t, original.U256.String(), decoded.U256.String())
		assert.NotNil(t, decoded.U256P)
		assert.Equal(t, original.U256P.String(), decoded.U256P.String())
		assert.Equal(t, original.Hash, decoded.Hash)
		assert.Equal(t, original.Nums, decoded.Nums)
		assert.Equal(t, original.Str, decoded.Str)
		assert.Equal(t, original.Bytes, decoded.Bytes)
		assert.Equal(t, original.Slice, decoded.Slice)
		assert.Equal(t, original.Inner, decoded.Inner)
	})
	
	t.Run("empty values", func(t *testing.T) {
		type EmptyValues struct {
			Before uint32   `ssz:"uint32"`
			Empty1 string   `ssz:"string"`
			Empty2 []byte   `ssz:"list" ssz-max:"100"`
			Empty3 []uint64 `ssz-max:"50"`
			After  uint32   `ssz:"uint32"`
		}
		
		original := EmptyValues{
			Before: 123,
			Empty1: "",
			Empty2: []byte{},
			Empty3: []uint64{},
			After:  456,
		}
		
		// Encode
		encoded, err := Marshal(original)
		require.NoError(t, err)
		
		// Decode
		var decoded EmptyValues
		err = Unmarshal(encoded, &decoded)
		require.NoError(t, err)
		
		// Compare
		assert.Equal(t, original, decoded)
	})
	
	t.Run("nested variable structs", func(t *testing.T) {
		type Level3 struct {
			Data []byte `ssz:"list" ssz-max:"32"`
		}
		
		type Level2 struct {
			Items []Level3 `ssz-max:"5"`
			Name  string   `ssz:"string"`
		}
		
		type Level1 struct {
			ID    uint32   `ssz:"uint32"`
			Inner []Level2 `ssz-max:"3"`
		}
		
		original := Level1{
			ID: 42,
			Inner: []Level2{
				{
					Items: []Level3{
						{Data: []byte{1, 2, 3}},
						{Data: []byte{4, 5, 6, 7}},
					},
					Name: "first",
				},
				{
					Items: []Level3{
						{Data: []byte{8, 9}},
					},
					Name: "second",
				},
			},
		}
		
		// Encode
		encoded, err := Marshal(original)
		require.NoError(t, err)
		
		// Decode
		var decoded Level1
		err = Unmarshal(encoded, &decoded)
		require.NoError(t, err)
		
		// Compare
		assert.Equal(t, original, decoded)
	})
	
	t.Run("max size values", func(t *testing.T) {
		type MaxSizes struct {
			BigSlice []uint64 `ssz-max:"1000"`
			BigBytes []byte   `ssz:"list" ssz-max:"2048"`
		}
		
		original := MaxSizes{
			BigSlice: make([]uint64, 1000),
			BigBytes: make([]byte, 2048),
		}
		
		// Fill with test data
		for i := range original.BigSlice {
			original.BigSlice[i] = uint64(i)
		}
		for i := range original.BigBytes {
			original.BigBytes[i] = byte(i % 256)
		}
		
		// Encode
		encoded, err := Marshal(original)
		require.NoError(t, err)
		
		// Decode
		var decoded MaxSizes
		err = Unmarshal(encoded, &decoded)
		require.NoError(t, err)
		
		// Compare
		assert.Equal(t, original, decoded)
	})
	
	t.Run("pointer fields", func(t *testing.T) {
		type WithPointers struct {
			Val1 *uint256.Int `ssz:"uint128"`
			Val2 *uint256.Int `ssz:"uint256"`
			Val3 uint256.Int  `ssz:"uint256"`
		}
		
		v1 := uint256.NewInt(100)
		v2 := uint256.NewInt(200)
		v3 := uint256.NewInt(300)
		
		original := WithPointers{
			Val1: v1,
			Val2: v2,
			Val3: *v3,
		}
		
		// Encode
		encoded, err := Marshal(original)
		require.NoError(t, err)
		
		// Decode
		var decoded WithPointers
		err = Unmarshal(encoded, &decoded)
		require.NoError(t, err)
		
		// Compare - pointers should be allocated
		require.NotNil(t, decoded.Val1)
		require.NotNil(t, decoded.Val2)
		assert.Equal(t, original.Val1.String(), decoded.Val1.String())
		assert.Equal(t, original.Val2.String(), decoded.Val2.String())
		assert.Equal(t, original.Val3.String(), decoded.Val3.String())
	})
}