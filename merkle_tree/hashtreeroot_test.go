package merkle_tree_test

import (
	"encoding/binary"
	"testing"

	"github.com/gfx-labs/ssz"
	"github.com/gfx-labs/ssz/merkle_tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashTreeRoot_SmallIntegers(t *testing.T) {
	t.Run("bool values", func(t *testing.T) {
		root1, err := merkle_tree.HashTreeRoot(true)
		require.NoError(t, err)
		assert.Equal(t, uint8(1), root1[0], "true should encode as 1")

		root2, err := merkle_tree.HashTreeRoot(false)
		require.NoError(t, err)
		assert.Equal(t, uint8(0), root2[0], "false should encode as 0")

		// Test multiple bool values
		root3, err := merkle_tree.HashTreeRoot(true, false, true, false)
		require.NoError(t, err)
		assert.NotEqual(t, [32]byte{}, root3, "Root should not be empty")
	})

	t.Run("uint8 values", func(t *testing.T) {
		var v1 uint8 = 0
		var v2 uint8 = 1
		var v3 uint8 = 127
		var v4 uint8 = 255

		root, err := merkle_tree.HashTreeRoot(v1, v2, v3, v4)
		require.NoError(t, err)
		assert.NotEqual(t, [32]byte{}, root, "Root should not be empty")

		// Test that different values produce different roots
		root1, err := merkle_tree.HashTreeRoot(v1)
		require.NoError(t, err)
		root2, err := merkle_tree.HashTreeRoot(v2)
		require.NoError(t, err)
		assert.NotEqual(t, root1, root2, "Different values should produce different roots")
	})

	t.Run("uint8 pointers", func(t *testing.T) {
		var v1 uint8 = 42
		var v2 uint8 = 100
		
		root1, err := merkle_tree.HashTreeRoot(&v1, &v2)
		require.NoError(t, err)
		
		root2, err := merkle_tree.HashTreeRoot(v1, v2)
		require.NoError(t, err)
		
		assert.Equal(t, root1, root2, "Pointer and value should produce same root")
	})

	t.Run("uint16 values", func(t *testing.T) {
		var v1 uint16 = 0
		var v2 uint16 = 256
		var v3 uint16 = 32767
		var v4 uint16 = 65535

		root, err := merkle_tree.HashTreeRoot(v1, v2, v3, v4)
		require.NoError(t, err)
		assert.NotEqual(t, [32]byte{}, root, "Root should not be empty")

		// Test that different values produce different roots
		root1, err := merkle_tree.HashTreeRoot(v1)
		require.NoError(t, err)
		root2, err := merkle_tree.HashTreeRoot(v2)
		require.NoError(t, err)
		assert.NotEqual(t, root1, root2, "Different values should produce different roots")
	})

	t.Run("uint16 pointers", func(t *testing.T) {
		var v1 uint16 = 1234
		var v2 uint16 = 5678
		
		root1, err := merkle_tree.HashTreeRoot(&v1, &v2)
		require.NoError(t, err)
		
		root2, err := merkle_tree.HashTreeRoot(v1, v2)
		require.NoError(t, err)
		
		assert.Equal(t, root1, root2, "Pointer and value should produce same root")
	})

	t.Run("uint32 values", func(t *testing.T) {
		var v1 uint32 = 0
		var v2 uint32 = 65536
		var v3 uint32 = 2147483647
		var v4 uint32 = 4294967295

		root, err := merkle_tree.HashTreeRoot(v1, v2, v3, v4)
		require.NoError(t, err)
		assert.NotEqual(t, [32]byte{}, root, "Root should not be empty")

		// Test that different values produce different roots
		root1, err := merkle_tree.HashTreeRoot(v1)
		require.NoError(t, err)
		root2, err := merkle_tree.HashTreeRoot(v2)
		require.NoError(t, err)
		assert.NotEqual(t, root1, root2, "Different values should produce different roots")
	})

	t.Run("uint32 pointers", func(t *testing.T) {
		var v1 uint32 = 12345678
		var v2 uint32 = 87654321
		
		root1, err := merkle_tree.HashTreeRoot(&v1, &v2)
		require.NoError(t, err)
		
		root2, err := merkle_tree.HashTreeRoot(v1, v2)
		require.NoError(t, err)
		
		assert.Equal(t, root1, root2, "Pointer and value should produce same root")
	})
}

func TestHashTreeRoot_MixedTypes(t *testing.T) {
	t.Run("mixed integer types", func(t *testing.T) {
		var v8 uint8 = 255
		var v16 uint16 = 65535
		var v32 uint32 = 4294967295
		var v64 uint64 = 18446744073709551615

		root, err := merkle_tree.HashTreeRoot(v8, v16, v32, v64)
		require.NoError(t, err)
		assert.NotEqual(t, [32]byte{}, root, "Root should not be empty")
	})

	t.Run("mixed pointers and values", func(t *testing.T) {
		var v8 uint8 = 10
		var v16 uint16 = 1000
		var v32 uint32 = 100000
		var v64 uint64 = 10000000

		root, err := merkle_tree.HashTreeRoot(&v8, v16, &v32, v64)
		require.NoError(t, err)
		assert.NotEqual(t, [32]byte{}, root, "Root should not be empty")
	})

	t.Run("integers with byte slices", func(t *testing.T) {
		var v8 uint8 = 42
		var v16 uint16 = 1337
		var v32 uint32 = 0xDEADBEEF
		var v64 uint64 = 0xCAFEBABE
		bytes := []byte{1, 2, 3, 4, 5}

		root, err := merkle_tree.HashTreeRoot(v8, v16, v32, v64, bytes)
		require.NoError(t, err)
		assert.NotEqual(t, [32]byte{}, root, "Root should not be empty")
	})
}

func TestHashTreeRoot_Encoding(t *testing.T) {
	t.Run("verify uint8 encoding", func(t *testing.T) {
		var v uint8 = 0xAB
		
		// Create expected leaves manually
		expectedLeaves := make([]byte, 32)
		expectedLeaves[0] = v
		
		// Get the root using HashTreeRoot
		root, err := merkle_tree.HashTreeRoot(v)
		require.NoError(t, err)
		
		// Verify the first byte matches our expectation
		assert.Equal(t, v, root[0], "First byte should match the uint8 value")
	})

	t.Run("verify uint16 little-endian encoding", func(t *testing.T) {
		var v uint16 = 0xABCD
		
		// Create expected encoding
		expectedBytes := make([]byte, 2)
		binary.LittleEndian.PutUint16(expectedBytes, v)
		
		// Get the root using HashTreeRoot
		root, err := merkle_tree.HashTreeRoot(v)
		require.NoError(t, err)
		
		// Verify the first two bytes match little-endian encoding
		assert.Equal(t, expectedBytes[0], root[0], "First byte should match little-endian encoding")
		assert.Equal(t, expectedBytes[1], root[1], "Second byte should match little-endian encoding")
	})

	t.Run("verify uint32 little-endian encoding", func(t *testing.T) {
		var v uint32 = 0xDEADBEEF
		
		// Create expected encoding
		expectedBytes := make([]byte, 4)
		binary.LittleEndian.PutUint32(expectedBytes, v)
		
		// Get the root using HashTreeRoot
		root, err := merkle_tree.HashTreeRoot(v)
		require.NoError(t, err)
		
		// Verify the first four bytes match little-endian encoding
		for i := 0; i < 4; i++ {
			assert.Equal(t, expectedBytes[i], root[i], "Byte %d should match little-endian encoding", i)
		}
	})
}

func TestHashTreeRoot_WithHashableSSZ(t *testing.T) {
	t.Run("mixed types with HashableSSZ", func(t *testing.T) {
		var v8 uint8 = 10
		var v16 uint16 = 100
		var v32 uint32 = 1000
		var v64 uint64 = 10000
		
		// Create a Prehash value
		prehash := ssz.Prehash{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16,
			17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
		
		root, err := merkle_tree.HashTreeRoot(v8, v16, v32, v64, &prehash)
		require.NoError(t, err)
		assert.NotEqual(t, [32]byte{}, root, "Root should not be empty")
	})
}

func BenchmarkHashTreeRoot_SmallIntegers(b *testing.B) {
	var v8 uint8 = 255
	var v16 uint16 = 65535
	var v32 uint32 = 4294967295
	var v64 uint64 = 18446744073709551615

	b.Run("uint8", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = merkle_tree.HashTreeRoot(v8)
		}
	})

	b.Run("uint16", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = merkle_tree.HashTreeRoot(v16)
		}
	})

	b.Run("uint32", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = merkle_tree.HashTreeRoot(v32)
		}
	})

	b.Run("uint64", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = merkle_tree.HashTreeRoot(v64)
		}
	})

	b.Run("mixed_types", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = merkle_tree.HashTreeRoot(v8, v16, v32, v64)
		}
	})
}