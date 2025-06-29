package merkle_tree

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/gfx-labs/ssz"
	"github.com/gfx-labs/ssz/merkle_tree/bufpool"
	"github.com/prysmaticlabs/gohashtree"
)

// HashTreeRoot returns the hash for a given schema of objects.
// IMPORTANT: DATA TYPE MUST IMPLEMENT ssz.HashableSSZ OR BE A SUPPORTED PRIMITIVE
// SUPPORTED PRIMITIVES: bool, uint8, *uint8, uint16, *uint16, uint32, *uint32, uint64, *uint64 and []byte
func HashTreeRoot(schema ...any) (out [32]byte, err error) {
	// Calculate the total number of leaves needed based on the schema length
	leaves := make([]byte, NextPowerOfTwo(uint64(len(schema)*32)))
	pos := 0

	// Iterate over each element in the schema
	for i, element := range schema {
		switch obj := element.(type) {
		case ssz.HashableSSZ:
			// If the element implements the HashableSSZ interface, calculate the SSZ hash and store it in the leaves
			root, err := obj.HashSSZ()
			if err != nil {
				return [32]byte{}, err
			}
			copy(leaves[pos:], root[:])
		case bool:
			if obj {
				leaves[pos] = 1
			}
		case uint8:
			// If the element is a uint8, encode it as little-endian and store it in the leaves
			leaves[pos] = obj
		case *uint8:
			// If the element is a pointer to uint8, dereference it and store it in the leaves
			leaves[pos] = *obj
		case uint16:
			// If the element is a uint16, encode it as little-endian and store it in the leaves
			binary.LittleEndian.PutUint16(leaves[pos:], obj)
		case *uint16:
			// If the element is a pointer to uint16, dereference it, encode it as little-endian, and store it in the leaves
			binary.LittleEndian.PutUint16(leaves[pos:], *obj)
		case uint32:
			// If the element is a uint32, encode it as little-endian and store it in the leaves
			binary.LittleEndian.PutUint32(leaves[pos:], obj)
		case *uint32:
			// If the element is a pointer to uint32, dereference it, encode it as little-endian, and store it in the leaves
			binary.LittleEndian.PutUint32(leaves[pos:], *obj)
		case uint64:
			// If the element is a uint64, encode it as little-endian and store it in the leaves
			binary.LittleEndian.PutUint64(leaves[pos:], obj)
		case *uint64:
			// If the element is a pointer to uint64, dereference it, encode it as little-endian, and store it in the leaves
			binary.LittleEndian.PutUint64(leaves[pos:], *obj)
		case []byte:
			// If the element is a byte slice
			if len(obj) < 32 {
				// If the slice is shorter than the length of a hash, copy the slice into the leaves
				copy(leaves[pos:], obj)
			} else {
				// If the slice is longer or equal to the length of a hash, calculate the hash of the slice and store it in the leaves
				root, err := BytesRoot(obj)
				if err != nil {
					return [32]byte{}, err
				}
				copy(leaves[pos:], root[:])
			}
		default:
			// If the element does not match any supported types, panic with an error message
			panic(fmt.Sprintf("Can't create TreeRoot: unsupported type %T at index %d", obj, i))
		}

		// Move the position pointer to the next leaf
		pos += 32
	}

	// Calculate the Merkle root from the flat leaves
	if err := ComputeMerkleRoot(leaves, leaves); err != nil {
		return [32]byte{}, err
	}

	// Convert the bytes of the resulting hash into a [32]byte and return it
	copy(out[:], leaves[:32])
	return out, nil
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
func MerkleProof(depth, proofIndex int, schema ...any) ([][32]byte, error) {
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
		schema = append(schema, make([]byte, 32))
	}

	for i := range depth {
		// Hash the left branch
		if proofIndex >= int(currentSizeDepth)/2 {
			proof[depth-i-1], err = HashTreeRoot(schema[0 : currentSizeDepth/2]...)
			if err != nil {
				return nil, err
			}
			schema = schema[currentSizeDepth/2:] // explore the right branch
			proofIndex -= int(currentSizeDepth) / 2
			currentSizeDepth /= 2
			continue
		}
		// Hash the right branch
		proof[depth-i-1], err = HashTreeRoot(schema[currentSizeDepth/2:]...)
		if err != nil {
			return nil, err
		}
		schema = schema[0 : currentSizeDepth/2] // explore the left branch
		currentSizeDepth /= 2
	}
	return proof, nil
}
