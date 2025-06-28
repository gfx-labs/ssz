package flexssz

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test the new fastssz-style tags
func TestFastSSZTags(t *testing.T) {
	t.Run("ssz-size for fixed byte array", func(t *testing.T) {
		type Test struct {
			Hash []byte `json:"hash" ssz-size:"32"`
		}
		
		// Validate struct
		MustPrecacheStructSSZInfo(Test{})
		
		// Encode
		original := Test{
			Hash: make([]byte, 32),
		}
		for i := range original.Hash {
			original.Hash[i] = byte(i)
		}
		
		encoded, err := EncodeStruct(original)
		require.NoError(t, err)
		assert.Len(t, encoded, 32)
		
		// Decode
		var decoded Test
		err = DecodeStruct(encoded, &decoded)
		require.NoError(t, err)
		assert.Equal(t, original, decoded)
	})
	
	t.Run("ssz-max for variable slice", func(t *testing.T) {
		type Test struct {
			Values []uint64 `json:"values" ssz-max:"100"`
		}
		
		// Validate struct
		MustPrecacheStructSSZInfo(Test{})
		
		// Encode
		original := Test{
			Values: []uint64{1, 2, 3, 4, 5},
		}
		
		encoded, err := EncodeStruct(original)
		require.NoError(t, err)
		
		// Decode
		var decoded Test
		err = DecodeStruct(encoded, &decoded)
		require.NoError(t, err)
		assert.Equal(t, original, decoded)
	})
	
	t.Run("ssz-size for multi-dimensional array", func(t *testing.T) {
		type Test struct {
			Roots [][]byte `json:"roots" ssz-size:"4,32"`
		}
		
		// Validate struct
		MustPrecacheStructSSZInfo(Test{})
		
		// Encode
		original := Test{
			Roots: make([][]byte, 4),
		}
		for i := range original.Roots {
			original.Roots[i] = make([]byte, 32)
			for j := range original.Roots[i] {
				original.Roots[i][j] = byte(i*32 + j)
			}
		}
		
		encoded, err := EncodeStruct(original)
		require.NoError(t, err)
		assert.Len(t, encoded, 4*32) // Fixed size
		
		// Decode
		var decoded Test
		err = DecodeStruct(encoded, &decoded)
		require.NoError(t, err)
		assert.Equal(t, original, decoded)
	})
	
	t.Run("ssz-max with ? for unlimited", func(t *testing.T) {
		type Test struct {
			Data []byte `json:"data" ssz-max:"?"`
		}
		
		// For now, we treat ? as no limit (0)
		MustPrecacheStructSSZInfo(Test{})
	})
	
	t.Run("complex beacon state example", func(t *testing.T) {
		type Fork struct {
			PreviousVersion []byte `json:"previous_version" ssz-size:"4"`
			CurrentVersion  []byte `json:"current_version" ssz-size:"4"`
			Epoch           uint64 `json:"epoch"`
		}
		
		type BeaconBlockHeader struct {
			Slot          uint64 `json:"slot"`
			ProposerIndex uint64 `json:"proposer_index"`
			ParentRoot    []byte `json:"parent_root" ssz-size:"32"`
			StateRoot     []byte `json:"state_root" ssz-size:"32"`
			BodyRoot      []byte `json:"body_root" ssz-size:"32"`
		}
		
		type Checkpoint struct {
			Epoch uint64 `json:"epoch"`
			Root  []byte `json:"root" ssz-size:"32"`
		}
		
		type BeaconState struct {
			GenesisTime           uint64             `json:"genesis_time"`
			GenesisValidatorsRoot []byte             `json:"genesis_validators_root" ssz-size:"32"`
			Slot                  uint64             `json:"slot"`
			Fork                  *Fork              `json:"fork"`
			LatestBlockHeader     *BeaconBlockHeader `json:"latest_block_header"`
			BlockRoots            [][]byte           `json:"block_roots" ssz-size:"8192,32"`
			StateRoots            [][]byte           `json:"state_roots" ssz-size:"8192,32"`
			JustificationBits     []byte             `json:"justification_bits" ssz-size:"1"`
			CurrentJustified      *Checkpoint        `json:"current_justified_checkpoint"`
		}
		
		// Validate struct
		MustPrecacheStructSSZInfo(BeaconState{})
		
		// Create test data
		original := BeaconState{
			GenesisTime:           12345,
			GenesisValidatorsRoot: make([]byte, 32),
			Slot:                  67890,
			Fork: &Fork{
				PreviousVersion: []byte{1, 0, 0, 0},
				CurrentVersion:  []byte{2, 0, 0, 0},
				Epoch:           100,
			},
			LatestBlockHeader: &BeaconBlockHeader{
				Slot:          67889,
				ProposerIndex: 42,
				ParentRoot:    make([]byte, 32),
				StateRoot:     make([]byte, 32),
				BodyRoot:      make([]byte, 32),
			},
			BlockRoots:        make([][]byte, 8192),
			StateRoots:        make([][]byte, 8192),
			JustificationBits: []byte{0xFF},
			CurrentJustified: &Checkpoint{
				Epoch: 99,
				Root:  make([]byte, 32),
			},
		}
		
		// Fill arrays with test data
		for i := 0; i < 32; i++ {
			original.GenesisValidatorsRoot[i] = byte(i)
		}
		
		// Initialize block roots and state roots
		for i := 0; i < 8192; i++ {
			original.BlockRoots[i] = make([]byte, 32)
			original.StateRoots[i] = make([]byte, 32)
			// Just fill first few for testing
			if i < 3 {
				for j := 0; j < 32; j++ {
					original.BlockRoots[i][j] = byte(i)
					original.StateRoots[i][j] = byte(i + 10)
				}
			}
		}
		
		// Encode
		encoded, err := EncodeStruct(original)
		require.NoError(t, err)
		
		// Decode
		var decoded BeaconState
		err = DecodeStruct(encoded, &decoded)
		require.NoError(t, err)
		
		// Compare
		assert.Equal(t, original.GenesisTime, decoded.GenesisTime)
		assert.Equal(t, original.GenesisValidatorsRoot, decoded.GenesisValidatorsRoot)
		assert.Equal(t, original.Slot, decoded.Slot)
		assert.Equal(t, original.Fork, decoded.Fork)
		assert.Equal(t, original.LatestBlockHeader, decoded.LatestBlockHeader)
		assert.Equal(t, original.JustificationBits, decoded.JustificationBits)
		assert.Equal(t, original.CurrentJustified, decoded.CurrentJustified)
		
		// Check first few roots
		for i := 0; i < 3; i++ {
			assert.Equal(t, original.BlockRoots[i], decoded.BlockRoots[i])
			assert.Equal(t, original.StateRoots[i], decoded.StateRoots[i])
		}
	})
	
	t.Run("validation errors", func(t *testing.T) {
		// Size mismatch
		t.Run("slice length mismatch", func(t *testing.T) {
			type Test struct {
				Data []byte `ssz-size:"32"`
			}
			
			test := Test{
				Data: make([]byte, 16), // Wrong size
			}
			
			_, err := EncodeStruct(test)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "does not match ssz-size")
		})
		
		// Both ssz-size and ssz-max
		t.Run("both size and max tags", func(t *testing.T) {
			type Test struct {
				Data []byte `ssz-size:"32" ssz-max:"100"`
			}
			
			err := PrecacheStructSSZInfo(Test{})
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "cannot use both ssz-size and ssz-max")
		})
		
		// Missing size/max on slice
		t.Run("missing size or max", func(t *testing.T) {
			type Test struct {
				Data []byte
			}
			
			err := PrecacheStructSSZInfo(Test{})
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "must have either ssz-size or ssz-max")
		})
	})
}

