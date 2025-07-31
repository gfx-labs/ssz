package flexssz_test

import (
	"encoding/hex"
	"testing"

	"github.com/gfx-labs/ssz/flexssz"
	"github.com/gfx-labs/ssz/merkle_tree"
	"github.com/stretchr/testify/require"
)

func TestMerkleCompareNested(t *testing.T) {

	type S struct {
		Foo uint8
	}

	type ps struct {
		Xs [][]S `ssz-max:"10,10"`
	}

	foos := func(xs ...uint8) []S {
		out := []S{}
		for i := range xs {
			out = append(out, S{Foo: xs[i]})
		}
		return out
	}

	state := &ps{
		Xs: [][]S{
			foos(1, 2, 3),
			foos(1, 2, 3, 4, 5, 6),
			foos(1, 2, 3, 4, 5),
		},
	}

	rootTree, err := flexssz.HashTreeRoot(state)
	require.NoError(t, err)

	// Manual merkleization of state
	// Step 1: Calculate merkle root for each inner list ([]S)
	// With ssz-max:"10,10", inner lists now have max size of 10

	// First inner list: [S{Foo: 1}, S{Foo: 2}, S{Foo: 3}]
	// Each S struct has one field (uint8), which gets padded to 32 bytes
	innerList1Chunks := make([][32]byte, 3)
	innerList1Chunks[0][0] = 1  // S{Foo: 1}
	innerList1Chunks[1][0] = 2  // S{Foo: 2}
	innerList1Chunks[2][0] = 3  // S{Foo: 3}
	
	// Merkleize with limit = 10 (from ssz-max:"10,10" second dimension)
	innerList1Data := make([]byte, 10*32) // 10 chunks (the limit)
	for i := 0; i < 3; i++ {
		copy(innerList1Data[i*32:], innerList1Chunks[i][:])
	}
	// The rest are zero padding
	
	var innerList1Root [32]byte
	err = merkle_tree.ComputeMerkleRootRange(innerList1Data, innerList1Root[:], 10, 0)
	require.NoError(t, err)
	
	// Mix in length for first inner list (length = 3)
	lengthRoot1 := merkle_tree.Uint64Root(3)
	innerList1RootWithLen := merkle_tree.Sha256(innerList1Root[:], lengthRoot1[:])
	innerList1Root = innerList1RootWithLen

	// Second inner list: [S{Foo: 1}, S{Foo: 2}, S{Foo: 3}, S{Foo: 4}, S{Foo: 5}, S{Foo: 6}]
	innerList2Chunks := make([][32]byte, 6)
	for i := 0; i < 6; i++ {
		innerList2Chunks[i][0] = uint8(i + 1)
	}
	
	// Merkleize with limit = 10
	innerList2Data := make([]byte, 10*32)
	for i := 0; i < 6; i++ {
		copy(innerList2Data[i*32:], innerList2Chunks[i][:])
	}
	
	var innerList2Root [32]byte
	err = merkle_tree.ComputeMerkleRootRange(innerList2Data, innerList2Root[:], 10, 0)
	require.NoError(t, err)
	
	// Mix in length
	lengthRoot2 := merkle_tree.Uint64Root(6)
	innerList2RootWithLen := merkle_tree.Sha256(innerList2Root[:], lengthRoot2[:])
	innerList2Root = innerList2RootWithLen

	// Third inner list: [S{Foo: 1}, S{Foo: 2}, S{Foo: 3}, S{Foo: 4}, S{Foo: 5}]
	innerList3Chunks := make([][32]byte, 5)
	for i := 0; i < 5; i++ {
		innerList3Chunks[i][0] = uint8(i + 1)
	}
	
	// Merkleize with limit = 10
	innerList3Data := make([]byte, 10*32)
	for i := 0; i < 5; i++ {
		copy(innerList3Data[i*32:], innerList3Chunks[i][:])
	}
	
	var innerList3Root [32]byte
	err = merkle_tree.ComputeMerkleRootRange(innerList3Data, innerList3Root[:], 10, 0)
	require.NoError(t, err)
	
	// Mix in length
	lengthRoot3 := merkle_tree.Uint64Root(5)
	innerList3RootWithLen := merkle_tree.Sha256(innerList3Root[:], lengthRoot3[:])
	innerList3Root = innerList3RootWithLen

	// Step 2: Calculate merkle root for the outer list ([][]S with max 10)
	// The outer list has 3 elements, each is the hash of an inner list
	outerListChunks := [][32]byte{
		innerList1Root,
		innerList2Root,
		innerList3Root,
	}

	// Need to merkleize with limit = 10 (from ssz-max:"10")
	// But we only have 3 elements, so we merkleize those 3 with appropriate padding
	outerListData := make([]byte, 10*32) // 10 chunks (the limit)
	for i := 0; i < 3; i++ {
		copy(outerListData[i*32:], outerListChunks[i][:])
	}
	// The rest are zero padding

	var outerListRoot [32]byte
	// Use ComputeMerkleRootRange with limit = 10
	err = merkle_tree.ComputeMerkleRootRange(outerListData, outerListRoot[:], 10, 0)
	require.NoError(t, err)

	// Mix in length for outer list (length = 3)
	outerLengthRoot := merkle_tree.Uint64Root(3)
	outerListRootWithLen := merkle_tree.Sha256(outerListRoot[:], outerLengthRoot[:])

	// Step 3: Calculate merkle root for the container (ps struct)
	// The container has 1 field: Xs
	var rawRoot [32]byte
	copy(rawRoot[:], outerListRootWithLen[:])

	require.Equal(t, hex.EncodeToString(rawRoot[:]), hex.EncodeToString(rootTree[:]))
}
