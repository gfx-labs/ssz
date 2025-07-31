package flexssz_test

import (
	"testing"

	"github.com/gfx-labs/ssz/flexssz"
	"github.com/stretchr/testify/require"
)

func TestMultidimensionalMaxTag(t *testing.T) {
	// Test 2D slice with ssz-max:"10,20"
	type Test2D struct {
		Data [][]uint64 `ssz-max:"10,20"`
	}

	test2d := &Test2D{
		Data: [][]uint64{
			{1, 2, 3},
			{4, 5, 6, 7},
			{8, 9},
		},
	}

	// Should encode and decode successfully
	encoded, err := flexssz.Marshal(test2d)
	require.NoError(t, err)

	decoded := &Test2D{}
	err = flexssz.Unmarshal(encoded, decoded)
	require.NoError(t, err)
	require.Equal(t, test2d.Data, decoded.Data)

	// Test 3D slice with ssz-max:"5,10,15"
	type Test3D struct {
		Data [][][]uint8 `ssz-max:"5,10,15"`
	}

	test3d := &Test3D{
		Data: [][][]uint8{
			{{1, 2}, {3, 4, 5}},
			{{6}, {7, 8}, {9, 10, 11}},
		},
	}

	// Should encode and decode successfully
	encoded3d, err := flexssz.Marshal(test3d)
	require.NoError(t, err)

	decoded3d := &Test3D{}
	err = flexssz.Unmarshal(encoded3d, decoded3d)
	require.NoError(t, err)
	require.Equal(t, test3d.Data, decoded3d.Data)

	// Test exceeding outer limit
	type TestLimit struct {
		Data [][]uint64 `ssz-max:"2,10"`
	}

	testLimit := &TestLimit{
		Data: [][]uint64{
			{1, 2, 3},
			{4, 5, 6},
			{7, 8, 9}, // This exceeds the outer limit of 2
		},
	}

	_, err = flexssz.Marshal(testLimit)
	require.Error(t, err)
	require.Contains(t, err.Error(), "exceeds limit")
}

func TestMultidimensionalMaxWithMerkleization(t *testing.T) {
	// Test that merkleization works correctly with multidimensional max tags
	type Inner struct {
		Value uint32
	}

	type Outer struct {
		Lists [][]Inner `ssz-max:"5,10"`
	}

	outer := &Outer{
		Lists: [][]Inner{
			{{Value: 1}, {Value: 2}},
			{{Value: 3}, {Value: 4}, {Value: 5}},
		},
	}

	// Compute hash tree root
	root1, err := flexssz.HashTreeRoot(outer)
	require.NoError(t, err)

	// Encode and decode to ensure consistency
	encoded, err := flexssz.Marshal(outer)
	require.NoError(t, err)

	decoded := &Outer{}
	err = flexssz.Unmarshal(encoded, decoded)
	require.NoError(t, err)

	// Hash tree root should be the same after round trip
	root2, err := flexssz.HashTreeRoot(decoded)
	require.NoError(t, err)
	require.Equal(t, root1, root2)
}