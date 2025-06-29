package merkle_tree

import (
	"errors"
	"fmt"

	"github.com/gfx-labs/ssz/merkle_tree/bufpool"
	"github.com/prysmaticlabs/gohashtree"
)

func MerklizeChunks(chunks [][32]byte, output []byte) (err error) {
	data := chunkedToSingle(chunks)
	return ComputeMerkleRootRange(data, output, NextPowerOfTwo(uint64((len(data)+31)/32)), 0)

}

func ComputeMerkleRoot(data []byte, output []byte) (err error) {
	if len(data) <= 32 {
		copy(output, data)
		return
	}
	return ComputeMerkleRootRange(data, output, NextPowerOfTwo(uint64((len(data)+31)/32)), 0)
}

func ComputeMerkleRootFromLevel(data []byte, output []byte, dataLength uint64, startLevel uint64) (err error) {
	if len(data) <= 32 {
		copy(output, data)
		return
	}
	return ComputeMerkleRootRange(data, output, NextPowerOfTwo(uint64((dataLength+31)/32)), uint64(startLevel))
}

func ComputeMerkleRootRange(data []byte, output []byte, leafLimit uint64, startLevel uint64) (err error) {
	if len(data)%32 != 0 {
		return errors.New("data length must be a multiple of 32")
	}
	// Get buffer from pool for reuse with enough capacity to avoid allocations
	poolBuffer := bufpool.Get(len(data) + 64)
	defer bufpool.Put(poolBuffer)

	// Initialize layer with input data. since we only ever make layer smaller, we rely on the fact that the runtime will not reallocate.
	// this is technically unsafe and relies on some golang internals, but the worst case is that it will trigger GC and fail to reuse the buffer if
	// somehow it gets replaced, so no big deal.
	layer := poolBuffer.B[:len(data)] // Set initial length to data size
	copy(layer, data)

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
	copy(output, layer[:32])
	return
}

// Merkle Proof computes the merkle proof for a given schema of objects.
func MerkleProof(depth, proofIndex int, schema ...[32]byte) ([][32]byte, error) {
	// Calculate the total number of leaves needed based on the schema length
	maxDepth := GetDepth(uint64(len(schema)))
	if PowerOf2(uint64(maxDepth)) != uint64(len(schema)) {
		maxDepth++
	}

	if depth != int(maxDepth) { // TODO: Add support for lower depths
		return nil, fmt.Errorf("depth is different than maximum depth, have %d, want %d", depth, maxDepth)
	}
	var err error
	proof := make([][32]byte, maxDepth)
	currentSizeDepth := PowerOf2(uint64(maxDepth))
	for len(schema) != int(currentSizeDepth) { // Augment the schema to be a power of 2
		schema = append(schema, [32]byte{})
	}

	for i := range depth {
		// Hash the left branch
		if proofIndex >= int(currentSizeDepth)/2 {
			err := MerklizeChunks(schema[0:currentSizeDepth/2], proof[depth-i-1][:])
			if err != nil {
				return nil, err
			}
			schema = schema[currentSizeDepth/2:] // explore the right branch
			proofIndex -= int(currentSizeDepth) / 2
			currentSizeDepth /= 2
			continue
		}
		// Hash the right branch
		err = MerklizeChunks(schema[currentSizeDepth/2:], proof[depth-i-1][:])
		if err != nil {
			return nil, err
		}
		schema = schema[0 : currentSizeDepth/2] // explore the left branch
		currentSizeDepth /= 2
	}
	return proof, nil
}
