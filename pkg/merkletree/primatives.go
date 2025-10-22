package merkle_tree

import (
	"encoding/binary"
)

// Uint64Root retrieves the root hash of a uint64 value by converting it to a byte array and returning it as a hash.
func Uint64Root(val uint64) (root [32]byte) {
	binary.LittleEndian.PutUint64(root[:], val)
	return root
}

func BytesRoot(b []byte) (out [32]byte, err error) {
	leafCount := NextPowerOfTwo(uint64((len(b) + 31) / 32))
	leaves := make([]byte, leafCount*32)
	copy(leaves, b)
	if err = ComputeMerkleRoot(leaves, leaves); err != nil {
		return [32]byte{}, err
	}
	copy(out[:], leaves)
	return
}

