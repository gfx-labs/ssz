package merkle_tree

import (
	"errors"
	"fmt"

	"github.com/gfx-labs/ssz/merkle_tree/bufpool"
	"github.com/prysmaticlabs/gohashtree"
)

func MerklizeChunks(chunks [][32]byte, rootOutput []byte) (err error) {
	data := chunkedToSingle(chunks)
	return ComputeMerkleRoot(data, rootOutput)

}

func ComputeMerkleRoot(inputData []byte, rootOutput []byte) (err error) {
	if len(inputData) <= 32 {
		copy(rootOutput, inputData)
		return
	}
	return ComputeMerkleRootRange(inputData, rootOutput, NextPowerOfTwo(uint64((len(inputData)+31)/32)), 0)
}

func ComputeMerkleRootFromLevel(inputData []byte, rootOutput []byte, dataLength uint64, startLevel uint64) (err error) {
	if len(inputData) <= 32 {
		copy(rootOutput, inputData)
		return
	}
	return ComputeMerkleRootRange(inputData, rootOutput, NextPowerOfTwo(uint64((dataLength+31)/32)), uint64(startLevel))
}

func ComputeMerkleRootRange(inputData []byte, rootOutput []byte, leafLimit uint64, startLevel uint64) (err error) {
	if len(inputData)%32 != 0 {
		return errors.New("data length must be a multiple of 32")
	}
	// Get buffer from pool for reuse with enough capacity to avoid allocations
	poolBuffer := bufpool.Get(len(inputData) + 64)
	defer bufpool.Put(poolBuffer)

	// Initialize layer with input data. since we only ever make layer smaller, we rely on the fact that the runtime will not reallocate.
	// this is technically unsafe and relies on some golang internals, but the worst case is that it will trigger GC and fail to reuse the buffer if
	// somehow it gets replaced, so no big deal.
	layer := poolBuffer.B[:len(inputData)] // Set initial length to data size
	copy(layer, inputData)

	for i := uint8(startLevel); i < GetDepth(leafLimit); i++ {
		layerLen := len(layer) / 32
		if layerLen%2 != 0 {
			// Append zero hash for padding - no allocation since we have capacity
			layer = append(layer, ZeroHashes[i][:]...)
			layerLen++
		}

		// Calculate output size for this iteration
		outputSize := (layerLen / 2) * 32

		// Hash in-place since output is always smaller than input
		if err := gohashtree.HashByteSlice(layer[:outputSize], layer); err != nil {
			return err
		}

		// Adjust layer size to the new smaller size
		layer = layer[:outputSize]
	}
	copy(rootOutput, layer[:32])
	return
}

// Merkle Proof computes the merkle proof for a given schema of objects.
func MerkleProof(treeDepth, leafIndex int, leaves ...[32]byte) ([][32]byte, error) {
	// Calculate the total number of leaves needed based on the leaves length
	maxDepth := GetDepth(uint64(len(leaves)))
	if PowerOf2(uint64(maxDepth)) != uint64(len(leaves)) {
		maxDepth++
	}

	if treeDepth != int(maxDepth) { // TODO: Add support for lower depths
		return nil, fmt.Errorf("depth is different than maximum depth, have %d, want %d", treeDepth, maxDepth)
	}
	var err error
	proof := make([][32]byte, maxDepth)
	currentSizeDepth := PowerOf2(uint64(maxDepth))
	for len(leaves) != int(currentSizeDepth) { // Augment the leaves to be a power of 2
		leaves = append(leaves, [32]byte{})
	}

	for i := range treeDepth {
		// Hash the left branch
		if leafIndex >= int(currentSizeDepth)/2 {
			err := MerklizeChunks(leaves[0:currentSizeDepth/2], proof[treeDepth-i-1][:])
			if err != nil {
				return nil, err
			}
			leaves = leaves[currentSizeDepth/2:] // explore the right branch
			leafIndex -= int(currentSizeDepth) / 2
			currentSizeDepth /= 2
			continue
		}
		// Hash the right branch
		err = MerklizeChunks(leaves[currentSizeDepth/2:], proof[treeDepth-i-1][:])
		if err != nil {
			return nil, err
		}
		leaves = leaves[0 : currentSizeDepth/2] // explore the left branch
		currentSizeDepth /= 2
	}
	return proof, nil
}
