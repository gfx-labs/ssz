package flexssz

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestGenesisHashTreeRoot(t *testing.T) {
	// Define the genesis struct
	type Genesis struct {
		GenesisValidatorsRoot [32]byte `ssz-size:"32"`
		GenesisTime           uint64   `ssz:"uint64"`
		GenesisForkVersion    [4]byte  `ssz-size:"4"`
	}

	// Test case 1: All zero values
	t.Run("all_zeros", func(t *testing.T) {
		genesis := &Genesis{
			GenesisValidatorsRoot: [32]byte{}, // All zeros
			GenesisTime:           0,
			GenesisForkVersion:    [4]byte{0x00, 0x00, 0x00, 0x00},
		}

		root, err := HashTreeRootStruct(genesis)
		if err != nil {
			t.Fatalf("Failed to calculate hash tree root: %v", err)
		}

		expectedHash := "db56114e00fdd4c1f85c892bf35ac9a89289aaecb1ebd0a96cde606a748b5d71"
		expectedBytes, err := hex.DecodeString(expectedHash)
		if err != nil {
			t.Fatalf("Failed to decode expected hash: %v", err)
		}

		if !bytes.Equal(root[:], expectedBytes) {
			t.Errorf("Hash mismatch:\nGot:      0x%x\nExpected: 0x%x", root, expectedBytes)
		} else {
			t.Logf("Genesis merkle root (correct): 0x%x", root)
		}
	})

	// Test case 2: Genesis time = 12345
	t.Run("genesis_time_12345", func(t *testing.T) {
		genesis := &Genesis{
			GenesisValidatorsRoot: [32]byte{}, // All zeros
			GenesisTime:           12345,
			GenesisForkVersion:    [4]byte{0x00, 0x00, 0x00, 0x00},
		}

		root, err := HashTreeRootStruct(genesis)
		if err != nil {
			t.Fatalf("Failed to calculate hash tree root: %v", err)
		}

		expectedHash := "f5b089f1f45195e02ab87fa5aa152eef5098e38a11e6d003811a63344d37b219"
		expectedBytes, err := hex.DecodeString(expectedHash)
		if err != nil {
			t.Fatalf("Failed to decode expected hash: %v", err)
		}

		if !bytes.Equal(root[:], expectedBytes) {
			t.Errorf("Hash mismatch:\nGot:      0x%x\nExpected: 0x%x", root, expectedBytes)
		} else {
			t.Logf("Genesis merkle root with time=12345 (correct): 0x%x", root)
		}
	})
}