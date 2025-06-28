package merkle_tree_test

import (
	"fmt"
	"testing"

	merkle_erigon "github.com/erigontech/erigon/cl/merkle_tree"
	"github.com/gfx-labs/ssz/merkle_tree"
	"github.com/stretchr/testify/require"
)

func TestComputeMerkleRootRange_CompareWithErigon(t *testing.T) {
	tests := []struct {
		name       string
		data       []byte
		leafLimit  uint64
		startLevel uint64
	}{
		{
			name:       "single leaf",
			data:       make([]byte, 32),
			leafLimit:  1,
			startLevel: 0,
		},
		{
			name: "two leaves",
			data: func() []byte {
				data := make([]byte, 64)
				data[0] = 1
				data[32] = 2
				return data
			}(),
			leafLimit:  2,
			startLevel: 0,
		},
		{
			name: "four leaves",
			data: func() []byte {
				data := make([]byte, 128)
				data[0] = 1
				data[32] = 2
				data[64] = 3
				data[96] = 4
				return data
			}(),
			leafLimit:  4,
			startLevel: 0,
		},
		{
			name: "eight leaves with start level 1",
			data: func() []byte {
				data := make([]byte, 256)
				for i := range 8 {
					data[i*32] = byte(i + 1)
				}
				return data
			}(),
			leafLimit:  8,
			startLevel: 1,
		},
		{
			name: "16 leaves with start level 2",
			data: func() []byte {
				data := make([]byte, 512)
				for i := range 16 {
					data[i*32] = byte(i)
				}
				return data
			}(),
			leafLimit:  16,
			startLevel: 2,
		},
		{
			name: "32 leaves",
			data: func() []byte {
				data := make([]byte, 1024)
				for i := range 32 {
					data[i*32] = byte(i % 256)
				}
				return data
			}(),
			leafLimit:  32,
			startLevel: 0,
		},
		{
			name: "odd number of leaves (5)",
			data: func() []byte {
				data := make([]byte, 160)
				for i := range 5 {
					data[i*32] = byte(i + 1)
				}
				return data
			}(),
			leafLimit:  8,
			startLevel: 0,
		},
		{
			name: "large limit with fewer leaves",
			data: func() []byte {
				data := make([]byte, 128)
				data[0] = 0xFF
				data[32] = 0xAA
				data[64] = 0x55
				data[96] = 0x33
				return data
			}(),
			leafLimit:  64,
			startLevel: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ourOutput := make([]byte, 32)
			erigonOutput := make([]byte, 32)

			ourErr := merkle_tree.ComputeMerkleRootRange(tt.data, ourOutput, tt.leafLimit, tt.startLevel)
			erigonErr := merkle_erigon.MerkleRootFromFlatFromIntermediateLevelWithLimit(tt.data, erigonOutput, int(tt.leafLimit), int(tt.startLevel))

			require.NoError(t, ourErr, "our implementation should not fail")
			require.NoError(t, erigonErr, "erigon implementation should not fail")

			require.Equal(t, erigonOutput, ourOutput,
				"merkle roots should match between our implementation and erigon\nOur: %x\nErigon: %x",
				ourOutput, erigonOutput)
		})
	}
}

func TestComputeMerkleRootRange_EdgeCasesWithErigon(t *testing.T) {
	t.Run("large leaf limit with small data", func(t *testing.T) {
		data := make([]byte, 64)
		data[0] = 1
		data[32] = 2

		ourOutput := make([]byte, 32)
		erigonOutput := make([]byte, 32)

		ourErr := merkle_tree.ComputeMerkleRootRange(data, ourOutput, 1<<10, 0)
		erigonErr := merkle_erigon.MerkleRootFromFlatFromIntermediateLevelWithLimit(data, erigonOutput, 1<<10, 0)

		require.NoError(t, ourErr)
		require.NoError(t, erigonErr)
		require.Equal(t, erigonOutput, ourOutput)
	})

	t.Run("maximum start level", func(t *testing.T) {
		data := make([]byte, 512)
		for i := range 16 {
			data[i*32] = byte(i)
		}

		leafLimit := uint64(16)
		depth := merkle_tree.GetDepth(leafLimit)
		maxStartLevel := uint64(depth - 1)

		ourOutput := make([]byte, 32)
		erigonOutput := make([]byte, 32)

		ourErr := merkle_tree.ComputeMerkleRootRange(data, ourOutput, leafLimit, maxStartLevel)
		erigonErr := merkle_erigon.MerkleRootFromFlatFromIntermediateLevelWithLimit(data, erigonOutput, int(leafLimit), int(maxStartLevel))

		require.NoError(t, ourErr)
		require.NoError(t, erigonErr)
		require.Equal(t, erigonOutput, ourOutput)
	})
}

func TestComputeMerkleRootRange_RandomDataWithErigon(t *testing.T) {
	testCases := []struct {
		leaves     int
		leafLimit  int
		startLevel int
	}{
		{1, 1, 0},
		{2, 2, 0},
		{3, 4, 0},
		{4, 4, 0},
		{7, 8, 0},
		{8, 8, 0},
		{15, 16, 0},
		{16, 16, 0},
		{8, 8, 1},
		{16, 16, 2},
		{32, 32, 3},
	}

	for _, tc := range testCases {
		t.Run(func() string {
			return fmt.Sprintf("leaves_%d_limit_%d_start_%d", tc.leaves, tc.leafLimit, tc.startLevel)
		}(), func(t *testing.T) {
			data := make([]byte, tc.leaves*32)
			for i := range tc.leaves {
				data[i*32] = byte(i*7 + 13)
				data[i*32+1] = byte(i*11 + 7)
				data[i*32+2] = byte(i*3 + 19)
			}

			ourOutput := make([]byte, 32)
			erigonOutput := make([]byte, 32)

			ourErr := merkle_tree.ComputeMerkleRootRange(data, ourOutput, uint64(tc.leafLimit), uint64(tc.startLevel))
			erigonErr := merkle_erigon.MerkleRootFromFlatFromIntermediateLevelWithLimit(data, erigonOutput, tc.leafLimit, tc.startLevel)

			require.NoError(t, ourErr, "our implementation should not fail")
			require.NoError(t, erigonErr, "erigon implementation should not fail")
			require.Equal(t, erigonOutput, ourOutput,
				"merkle roots should match\nOur: %x\nErigon: %x",
				ourOutput, erigonOutput)
		})
	}
}

func BenchmarkComputeMerkleRootRange_VsErigon(b *testing.B) {
	data := make([]byte, 1024*32)
	for i := range 1024 {
		data[i*32] = byte(i % 256)
	}

	b.Run("our_implementation", func(b *testing.B) {
		output := make([]byte, 32)
		for b.Loop() {
			_ = merkle_tree.ComputeMerkleRootRange(data, output, 1024, 0)
		}
	})

	b.Run("erigon_implementation", func(b *testing.B) {
		output := make([]byte, 32)
		for b.Loop() {
			_ = merkle_erigon.MerkleRootFromFlatFromIntermediateLevelWithLimit(data, output, 1024, 0)
		}
	})
}
