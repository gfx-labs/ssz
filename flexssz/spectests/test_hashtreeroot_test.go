package spectests

import (
	"encoding/hex"
	"testing"

	"github.com/gfx-labs/ssz/flexssz"
)

func TestHashTreeRoot(t *testing.T) {
	// Test with struct
	t.Run("Struct", func(t *testing.T) {
		type SimpleStruct struct {
			A uint64 `json:"a"`
			B uint32 `json:"b"`
		}

		s := &SimpleStruct{A: 12345, B: 67890}

		// Test new HashTreeRoot
		hash, err := flexssz.HashTreeRoot(s)
		if err != nil {
			t.Fatalf("Failed to hash struct: %v", err)
		}
		t.Logf("✓ Struct hash: 0x%s", hex.EncodeToString(hash[:]))
	})

	// Test with basic types
	t.Run("Uint64", func(t *testing.T) {
		val := uint64(12345678901234567890)

		hash, err := flexssz.HashTreeRoot(val)
		if err != nil {
			t.Fatalf("Failed to hash uint64: %v", err)
		}

		t.Logf("✓ uint64 hash: 0x%s", hex.EncodeToString(hash[:]))
	})

	// Test with slice
	t.Run("Slice", func(t *testing.T) {
		// For slices to work properly, they need to be in a struct with tags
		// Direct slice hashing would need type info
		type SliceContainer struct {
			Items []uint64 `json:"items" ssz-max:"100"`
		}

		s := &SliceContainer{Items: []uint64{1, 2, 3, 4, 5}}

		hash, err := flexssz.HashTreeRoot(s)
		if err != nil {
			t.Fatalf("Failed to hash slice container: %v", err)
		}

		t.Logf("✓ Slice container hash: 0x%s", hex.EncodeToString(hash[:]))
	})

	// Test with byte array
	t.Run("ByteArray", func(t *testing.T) {
		var arr [32]byte
		for i := range arr {
			arr[i] = byte(i)
		}

		hash, err := flexssz.HashTreeRoot(arr)
		if err != nil {
			t.Fatalf("Failed to hash byte array: %v", err)
		}

		// For a 32-byte array, the hash should be the array itself
		if hash != arr {
			t.Errorf("32-byte array hash should equal the array itself")
		}

		t.Logf("✓ Byte array hash: 0x%s", hex.EncodeToString(hash[:8])+"...")
	})
}

