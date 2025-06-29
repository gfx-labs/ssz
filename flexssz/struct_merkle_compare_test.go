package flexssz_test

import (
	"encoding/hex"
	"testing"

	"github.com/gfx-labs/ssz/flexssz"
	dynssz "github.com/pk910/dynamic-ssz"
	"github.com/stretchr/testify/require"
)

func TestMerkleCompare(t *testing.T) {

	type s struct {
		BlockRoots [][]byte `json:"block_roots" ssz-size:"8192,32"`
	}

	state := &s{
		BlockRoots: make([][]byte, 8192),
	}
	for i := range len(state.BlockRoots) {
		state.BlockRoots[i] = make([]byte, 32)
	}

	//typeInfo, err := flexssz.GetTypeInfo(reflect.TypeOf(state), nil)
	//require.NoError(t, err)
	//spew.Dump(typeInfo)

	rootTree, err := flexssz.HashTreeRoot(state)
	require.NoError(t, err)
	dzRoot, err := dynssz.NewDynSsz(nil).HashTreeRoot(state)
	require.NoError(t, err)

	require.Equal(t, hex.EncodeToString(dzRoot[:]), hex.EncodeToString(rootTree[:]))
}
