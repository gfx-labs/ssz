package flexssz

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContainerTag(t *testing.T) {
	t.Run("container tag on struct", func(t *testing.T) {
		type Inner struct {
			Value uint32 `ssz:"uint32"`
		}
		
		type Outer struct {
			ID    uint64 `ssz:"uint64"`
			Data  Inner  `ssz:"container"`  // Explicit container tag
			Count uint16 `ssz:"uint16"`
		}
		
		err := PrecacheStructSSZInfo(Outer{})
		require.NoError(t, err)
		
		// Test encoding
		s := Outer{
			ID:    12345,
			Data:  Inner{Value: 42},
			Count: 999,
		}
		
		encoded, err := EncodeStruct(s)
		require.NoError(t, err)
		assert.NotEmpty(t, encoded)
		
		// Verify by decoding
		d := NewDecoder(encoded)
		
		id, err := d.ReadUint64()
		require.NoError(t, err)
		assert.Equal(t, uint64(12345), id)
		
		value, err := d.ReadUint32()
		require.NoError(t, err)
		assert.Equal(t, uint32(42), value)
		
		count, err := d.ReadUint16()
		require.NoError(t, err)
		assert.Equal(t, uint16(999), count)
	})
	
	t.Run("container tag on non-struct fails", func(t *testing.T) {
		type Invalid struct {
			Data uint64 `ssz:"container"`  // Can't use container on non-struct
		}
		
		err := PrecacheStructSSZInfo(Invalid{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ssz tag 'container' requires struct or pointer to struct type")
	})
	
	t.Run("auto-detected container", func(t *testing.T) {
		type Inner struct {
			A uint32 `ssz:"uint32"`
			B bool   `ssz:"bool"`
		}
		
		type Outer struct {
			Before uint64
			Nested Inner   // Auto-detected as container
			After  uint16
		}
		
		err := PrecacheStructSSZInfo(Outer{})
		require.NoError(t, err)
		
		// Test encoding
		s := Outer{
			Before: 100,
			Nested: Inner{A: 200, B: true},
			After:  300,
		}
		
		encoded, err := EncodeStruct(s)
		require.NoError(t, err)
		assert.NotEmpty(t, encoded)
	})
	
	t.Run("nested containers with variable fields", func(t *testing.T) {
		type InnerVariable struct {
			Fixed uint32 `ssz:"uint32"`
			Data  []byte `ssz:"list" ssz-max:"100"`
		}
		
		type OuterVariable struct {
			ID       uint64        `ssz:"uint64"`
			Variable InnerVariable `ssz:"container"`
			End      uint16        `ssz:"uint16"`
		}
		
		err := PrecacheStructSSZInfo(OuterVariable{})
		require.NoError(t, err)
		
		// Test encoding
		s := OuterVariable{
			ID: 1000,
			Variable: InnerVariable{
				Fixed: 2000,
				Data:  []byte("test data"),
			},
			End: 3000,
		}
		
		encoded, err := EncodeStruct(s)
		require.NoError(t, err)
		assert.NotEmpty(t, encoded)
		
		// The encoding should handle the variable container properly
		// with offsets for the variable field
	})
}