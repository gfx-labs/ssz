package merkle_tree_test

import (
	"testing"

	"github.com/gfx-labs/ssz/merkle_tree"
	"github.com/stretchr/testify/require"
)

func getExpectedRoot(testBuffer []byte) [32]byte {
	var root [32]byte
	merkle_tree.ComputeMerkleRoot(testBuffer, root[:])
	return root
}

func getExpectedRootWithLimit(testBuffer []byte, limit int) [32]byte {
	var root [32]byte
	merkle_tree.ComputeMerkleRootRange(testBuffer, root[:], uint64(limit), 0)
	return root
}

func TestPowerOf2MerkleTree(t *testing.T) {
	mt := merkle_tree.MerkleTree{}
	testBuffer := make([]byte, 4*32)
	testBuffer[0] = 1
	testBuffer[32] = 2
	testBuffer[64] = 3
	testBuffer[96] = 9
	mt.Initialize(4, 6, func(idx int, out []byte) {
		copy(out, testBuffer[idx*32:(idx+1)*32])
	}, nil)
	expectedRoot1 := getExpectedRoot(testBuffer)
	require.Equal(t, expectedRoot1, mt.ComputeRoot())
	testBuffer[64] = 4
	require.Equal(t, expectedRoot1, mt.ComputeRoot())
	mt.MarkLeafAsDirty(2)
	expectedRoot2 := getExpectedRoot(testBuffer)
	require.Equal(t, expectedRoot2, mt.ComputeRoot())
	testBuffer[64] = 3
	mt.MarkLeafAsDirty(2)
	require.Equal(t, expectedRoot1, mt.ComputeRoot())

}

func TestMerkleTreeAppendLeaf(t *testing.T) {
	mt := merkle_tree.MerkleTree{}
	testBuffer := make([]byte, 4*32)
	testBuffer[0] = 1
	testBuffer[32] = 2
	testBuffer[64] = 3
	testBuffer[96] = 9
	mt.Initialize(4, 6, func(idx int, out []byte) {
		copy(out, testBuffer[idx*32:(idx+1)*32])
	}, nil)
	// Test AppendLeaf
	mt.AppendLeaf()
	testBuffer = append(testBuffer, make([]byte, 4*32)...)
	testBuffer[128] = 5
	expectedRoot1 := getExpectedRoot(testBuffer)
	require.Equal(t, expectedRoot1, mt.ComputeRoot())
	// adding 3 more empty leaves should not change the root
	mt.AppendLeaf()
	mt.AppendLeaf()
	mt.AppendLeaf()
	require.Equal(t, expectedRoot1, mt.ComputeRoot())
}

func TestMerkleTreeRootEmpty(t *testing.T) {
	mt := merkle_tree.MerkleTree{}
	mt.Initialize(0, 6, func(idx int, out []byte) {
		return
	}, nil)
	require.Equal(t, [32]byte{}, mt.ComputeRoot())
}

func TestMerkleTreeRootSingleElement(t *testing.T) {
	mt := merkle_tree.MerkleTree{}
	testBuffer := make([]byte, 32)
	testBuffer[0] = 1
	mt.Initialize(1, 6, func(idx int, out []byte) {
		copy(out, testBuffer)
	}, nil)
	require.Equal(t, [32]byte{1}, mt.ComputeRoot())
}

func TestMerkleTreeAppendLeafWithLowMaxDepth(t *testing.T) {
	mt := merkle_tree.MerkleTree{}
	testBuffer := make([]byte, 4*32)
	testBuffer[0] = 1
	testBuffer[32] = 2
	testBuffer[64] = 3
	testBuffer[96] = 9
	mt.Initialize(4, 2, func(idx int, out []byte) {
		copy(out, testBuffer[idx*32:(idx+1)*32])
	}, nil)
	// Test AppendLeaf
	mt.AppendLeaf()
	testBuffer = append(testBuffer, make([]byte, 4*32)...)
	testBuffer[128] = 5
	expectedRoot := getExpectedRoot(testBuffer)
	require.Equal(t, expectedRoot, mt.ComputeRoot())
	// adding 3 more empty leaves should not change the root
	mt.AppendLeaf()
	mt.AppendLeaf()
	mt.AppendLeaf()
	require.Equal(t, expectedRoot, mt.ComputeRoot())
}

func TestMerkleTree17Elements(t *testing.T) {
	mt := merkle_tree.MerkleTree{}
	testBuffer := make([]byte, 17*32)
	testBuffer[0] = 1
	testBuffer[32] = 2
	testBuffer[64] = 3
	testBuffer[96] = 9
	testBuffer[128] = 5
	mt.Initialize(17, 2, func(idx int, out []byte) {
		copy(out, testBuffer[idx*32:(idx+1)*32])
	}, nil)
	// Test AppendLeaf
	expectedRoot := getExpectedRoot(testBuffer)
	require.Equal(t, expectedRoot, mt.ComputeRoot())
}

func TestMerkleTreeAppendLeafWithLowMaxDepthAndLimitAndTestWR(t *testing.T) {
	mt := merkle_tree.MerkleTree{}
	testBuffer := make([]byte, 4*32)
	testBuffer[0] = 1
	testBuffer[32] = 2
	testBuffer[64] = 3
	testBuffer[96] = 9
	lm := uint64(1 << 12)
	mt.Initialize(4, 2, func(idx int, out []byte) {
		copy(out, testBuffer[idx*32:(idx+1)*32])
	}, &lm)
	// Test AppendLeaf
	mt.AppendLeaf()
	testBuffer = append(testBuffer, make([]byte, 4*32)...)
	testBuffer[128] = 5
	expectedRoot := getExpectedRootWithLimit(testBuffer, int(lm))
	require.Equal(t, expectedRoot, mt.ComputeRoot())
}
