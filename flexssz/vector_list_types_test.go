package flexssz

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVectorListTypes(t *testing.T) {
	t.Run("comprehensive vector and list usage", func(t *testing.T) {
		type TestStruct struct {
			// Basic types
			Count  uint32 `ssz:"uint32"`
			Active bool   `ssz:"bool"`
			
			// Vectors (fixed-size arrays)
			BlockHash    [32]byte     `ssz:"vector"`    // Explicit vector tag
			Signature    [96]byte                       // Auto-detected as vector
			Uint256Val   uint256.Int  `ssz:"uint256"`  // Special handling
			
			// Lists (dynamic slices with limits)
			Validators   []uint64     `ssz-max:"10000"`              // Auto-detected as list
			Messages     []string     `ssz:"list" ssz-max:"100"`     
			ByteData     []byte       `ssz:"list" ssz-max:"2048"`    // Explicit list tag for bytes
			
			// Nested structures
			Entries []struct {
				Key   [20]byte `ssz:"vector"`
				Value []byte   `ssz:"list" ssz-max:"256"`
			} `ssz-max:"1000"`
		}
		
		// Validate struct
		err := PrecacheStructSSZInfo(TestStruct{})
		require.NoError(t, err)
		
		// Create and encode
		s := TestStruct{
			Count:      42,
			Active:     true,
			Validators: []uint64{1, 2, 3, 4, 5},
			Messages:   []string{"hello", "world"},
			ByteData:   []byte("test data"),
		}
		
		// Set some fixed values
		copy(s.BlockHash[:], []byte("block-hash-example"))
		copy(s.Signature[:], []byte("signature-example"))
		s.Uint256Val.SetUint64(123456)
		
		// Add some entries
		s.Entries = []struct {
			Key   [20]byte `ssz:"vector"`
			Value []byte   `ssz:"list" ssz-max:"256"`
		}{
			{Value: []byte("value1")},
			{Value: []byte("value2")},
		}
		copy(s.Entries[0].Key[:], []byte("key1"))
		copy(s.Entries[1].Key[:], []byte("key2"))
		
		encoded, err := EncodeStruct(s)
		require.NoError(t, err)
		assert.NotEmpty(t, encoded)
		
		// Verify encoding contains expected data
		d := NewDecoder(encoded)
		
		// Read fixed fields
		count, err := d.ReadUint32()
		require.NoError(t, err)
		assert.Equal(t, uint32(42), count)
		
		active, err := d.ReadBool()
		require.NoError(t, err)
		assert.True(t, active)
	})
	
	t.Run("vector tag validation", func(t *testing.T) {
		// Valid: vector on array
		type ValidVector struct {
			Data [10]uint32 `ssz:"vector"`
		}
		err := PrecacheStructSSZInfo(ValidVector{})
		require.NoError(t, err)
		
		// Invalid: vector on slice
		type InvalidVector struct {
			Data []uint32 `ssz:"vector"`
		}
		err = PrecacheStructSSZInfo(InvalidVector{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ssz tag 'vector' requires array type")
	})
	
	t.Run("list tag validation", func(t *testing.T) {
		// Valid: list on slice with limit
		type ValidList struct {
			Data []uint32 `ssz:"list" ssz-max:"100"`
		}
		err := PrecacheStructSSZInfo(ValidList{})
		require.NoError(t, err)
		
		// Invalid: list on array
		type InvalidList struct {
			Data [10]uint32 `ssz:"list"`
		}
		err = PrecacheStructSSZInfo(InvalidList{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ssz tag 'list' requires slice type")
		
		// Invalid: list without limit
		type ListNoLimit struct {
			Data []uint32 `ssz:"list"`
		}
		err = PrecacheStructSSZInfo(ListNoLimit{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "slice types must have either ssz-size or ssz-max tag")
	})
	
	t.Run("auto-detection", func(t *testing.T) {
		type AutoDetect struct {
			// Arrays auto-detect as vector
			Hash1 [32]byte
			Hash2 [32]byte `ssz:"vector"` // Explicit is also fine
			
			// Slices auto-detect as list (but need limit)
			Data1 []byte   `ssz-max:"100"`
			Data2 []byte   `ssz:"list" ssz-max:"100"` // Explicit is also fine
		}
		
		err := PrecacheStructSSZInfo(AutoDetect{})
		require.NoError(t, err)
	})
}