package spectests

import (
	"bytes"
	"compress/gzip"
	"encoding/hex"
	"io"
	"os"
	"testing"

	"github.com/ferranbt/fastssz/spectests"
	"github.com/gfx-labs/ssz/flexssz"
	dynssz "github.com/pk910/dynamic-ssz"
	"github.com/stretchr/testify/require"
)

func TestFieldByFieldComparison(t *testing.T) {
	// Read the fixture file
	fixturePath := "_fixtures/beacon_state_bellatrix.ssz.gz"

	file, err := os.Open(fixturePath)
	if err != nil {
		t.Fatalf("Failed to open fixture file: %v", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gzReader.Close()

	originalData, err := io.ReadAll(gzReader)
	if err != nil {
		t.Fatalf("Failed to read fixture data: %v", err)
	}

	// Unmarshal with both implementations
	fastState := &spectests.BeaconStateBellatrix{}
	if err := fastState.UnmarshalSSZ(originalData); err != nil {
		t.Fatalf("Failed to unmarshal with fastssz: %v", err)
	}

	ourState := &BeaconStateBellatrix{}
	if err := flexssz.Unmarshal(originalData, ourState); err != nil {
		t.Fatalf("Failed to unmarshal with flexssz: %v", err)
	}

	// Compare each field's hash
	t.Run("GenesisTime", func(t *testing.T) {
		// For fastssz, we need to create a wrapper struct or use the field directly
		fastVal := fastState.GenesisTime
		fastHash := hashUint64(fastVal)

		ourHash, err := flexssz.HashTreeRoot(struct{ Val uint64 }{Val: ourState.GenesisTime})
		if err != nil {
			t.Fatalf("Failed to hash: %v", err)
		}
		compareHashes(t, "GenesisTime", fastHash, ourHash)
	})

	t.Run("GenesisValidatorsRoot", func(t *testing.T) {
		// For byte arrays, hash directly
		var fastArr, ourArr [32]byte
		copy(fastArr[:], fastState.GenesisValidatorsRoot)
		copy(ourArr[:], ourState.GenesisValidatorsRoot)
		fastHash := hashBytes32(fastArr)
		ourHash := hashBytes32(ourArr)
		compareHashes(t, "GenesisValidatorsRoot", fastHash, ourHash)
	})

	t.Run("Slot", func(t *testing.T) {
		fastHash := hashUint64(fastState.Slot)
		ourHash, err := flexssz.HashTreeRoot(struct{ Val uint64 }{Val: ourState.Slot})
		if err != nil {
			t.Fatalf("Failed to hash: %v", err)
		}
		compareHashes(t, "Slot", fastHash, ourHash)
	})

	t.Run("Fork", func(t *testing.T) {
		fastHash, _ := fastState.Fork.HashTreeRoot()
		ourHash, _ := flexssz.HashTreeRoot(ourState.Fork)
		compareHashes(t, "Fork", fastHash, ourHash)
	})

	t.Run("LatestBlockHeader", func(t *testing.T) {
		fastHash, _ := fastState.LatestBlockHeader.HashTreeRoot()
		ourHash, _ := flexssz.HashTreeRoot(ourState.LatestBlockHeader)
		compareHashes(t, "LatestBlockHeader", fastHash, ourHash)
	})

	t.Run("BlockRoots", func(t *testing.T) {
		// For BlockRoots, we need to compare the entire vector hash
		// fastssz uses [][32]byte format
		// We can't easily hash just the BlockRoots field from fastssz without the full state
		// So let's compare the marshaled data instead

		t.Logf("BlockRoots count - fast: %d, our: %d", len(fastState.BlockRoots), len(ourState.BlockRoots))

		// Check first few individual roots match
		if len(fastState.BlockRoots) > 0 {
			for i := 0; i < 3 && i < len(fastState.BlockRoots); i++ {
				fastRoot := fastState.BlockRoots[i]
				ourRoot := ourState.BlockRoots[i]
				match := bytes.Equal(fastRoot[:], ourRoot[:])
				t.Logf("BlockRoots[%d] match: %v", i, match)
				if !match {
					t.Logf("  fast: %x", fastRoot[:8])
					t.Logf("  our:  %x", ourRoot[:8])
				}
			}
		}
	})

	t.Run("Validators", func(t *testing.T) {
		t.Logf("Validators count - fast: %d, our: %d", len(fastState.Validators), len(ourState.Validators))

		// Check first validator if exists
		if len(fastState.Validators) > 0 && len(ourState.Validators) > 0 {
			fastValHash, _ := fastState.Validators[0].HashTreeRoot()
			ourValHash, _ := flexssz.HashTreeRoot(ourState.Validators[0])
			compareHashes(t, "Validators[0]", fastValHash, ourValHash)

			// Check individual fields of first validator
			t.Logf("Validator[0].Pubkey match: %v", bytes.Equal(fastState.Validators[0].Pubkey, ourState.Validators[0].Pubkey))
			t.Logf("Validator[0].WithdrawalCredentials match: %v",
				bytes.Equal(fastState.Validators[0].WithdrawalCredentials, ourState.Validators[0].WithdrawalCredentials))
			t.Logf("Validator[0].EffectiveBalance - fast: %d, our: %d",
				fastState.Validators[0].EffectiveBalance, ourState.Validators[0].EffectiveBalance)
			t.Logf("Validator[0].Slashed - fast: %v, our: %v",
				fastState.Validators[0].Slashed, ourState.Validators[0].Slashed)
		}
	})

	t.Run("Balances", func(t *testing.T) {
		t.Logf("Balances count - fast: %d, our: %d", len(fastState.Balances), len(ourState.Balances))

		// Check first few balances
		for i := 0; i < 5 && i < len(fastState.Balances) && i < len(ourState.Balances); i++ {
			if fastState.Balances[i] != ourState.Balances[i] {
				t.Logf("Balance[%d] mismatch - fast: %d, our: %d", i, fastState.Balances[i], ourState.Balances[i])
			}
		}
	})

	t.Run("Eth1DataVotes", func(t *testing.T) {
		t.Logf("Eth1DataVotes length - fast: %d, our: %d", len(fastState.Eth1DataVotes), len(ourState.Eth1DataVotes))

		// Check if first vote matches (if any)
		if len(fastState.Eth1DataVotes) > 0 && len(ourState.Eth1DataVotes) > 0 {
			fastVoteHash, _ := fastState.Eth1DataVotes[0].HashTreeRoot()
			ourVoteHash, _ := flexssz.HashTreeRoot(ourState.Eth1DataVotes[0])
			compareHashes(t, "Eth1DataVotes[0]", fastVoteHash, ourVoteHash)
		}
	})

	t.Run("PreviousEpochParticipation", func(t *testing.T) {
		t.Logf("PreviousEpochParticipation length - fast: %d, our: %d",
			len(fastState.PreviousEpochParticipation), len(ourState.PreviousEpochParticipation))

		// Check first few values
		for i := 0; i < 5 && i < len(fastState.PreviousEpochParticipation) && i < len(ourState.PreviousEpochParticipation); i++ {
			if fastState.PreviousEpochParticipation[i] != ourState.PreviousEpochParticipation[i] {
				t.Logf("PreviousEpochParticipation[%d] mismatch - fast: %d, our: %d",
					i, fastState.PreviousEpochParticipation[i], ourState.PreviousEpochParticipation[i])
			}
		}
	})

	t.Run("CurrentEpochParticipation", func(t *testing.T) {
		t.Logf("CurrentEpochParticipation length - fast: %d, our: %d",
			len(fastState.CurrentEpochParticipation), len(ourState.CurrentEpochParticipation))
	})

	t.Run("InactivityScores", func(t *testing.T) {
		t.Logf("InactivityScores length - fast: %d, our: %d",
			len(fastState.InactivityScores), len(ourState.InactivityScores))
	})

	t.Run("HistoricalRoots", func(t *testing.T) {
		require.Equal(t, len(fastState.HistoricalRoots), len(ourState.HistoricalRoots),
			"HistoricalRoots length mismatch")

		// Calculate merkle root using flexssz
		// Create a struct with just the HistoricalRoots field
		type HistoricalRootsOnly struct {
			Roots [][32]byte `ssz-max:"16777216"`
		}

		roots := make([][32]byte, len(ourState.HistoricalRoots))
		for i := range ourState.HistoricalRoots {
			copy(roots[i][:], ourState.HistoricalRoots[i][:])
		}

		testStruct := &HistoricalRootsOnly{Roots: roots}
		theirRoot, err := dynssz.NewDynSsz(nil).HashTreeRoot(testStruct)
		require.NoError(t, err, "Failed to calculate dynamic-ssz merkle root for HistoricalRoots")
		ourRoot, err := flexssz.HashTreeRoot(testStruct)
		require.NoError(t, err, "Failed to calculate flexssz merkle root for HistoricalRoots")

		// Compare the merkle roots
		t.Logf("HistoricalRoots merkle root comparison:")
		t.Logf("  dynamic-ssz: 0x%s", hex.EncodeToString(theirRoot[:]))
		t.Logf("  flexssz:     0x%s", hex.EncodeToString(ourRoot[:]))

		require.Equal(t, theirRoot, ourRoot, "HistoricalRoots merkle roots do not match between dynamic-ssz and flexssz")

		if len(ourState.HistoricalRoots) > 0 {
			t.Logf("✓ HistoricalRoots merkle roots match! (contains %d elements)", len(ourState.HistoricalRoots))
		} else {
			t.Log("✓ HistoricalRoots merkle roots match! (empty list)")
		}
	})

	// Test specific problematic fields more deeply
	t.Run("DetailedFieldAnalysis", func(t *testing.T) {
		// Let's calculate the state root manually by field order
		// to see where the divergence happens

		// The BeaconStateBellatrix fields in order are:
		// 1. GenesisTime (uint64)
		// 2. GenesisValidatorsRoot ([32]byte)
		// 3. Slot (uint64)
		// 4. Fork (container)
		// 5. LatestBlockHeader (container)
		// 6. BlockRoots (Vector[Root, 8192])
		// 7. StateRoots (Vector[Root, 8192])
		// 8. HistoricalRoots (List[Root])
		// ... and so on

		// Let's check if the issue is with vectors of roots
		t.Logf("=== Vector Analysis ===")
		t.Logf("BlockRoots is Vector[Root, 8192] where Root = [32]byte")
		t.Logf("StateRoots is Vector[Root, 8192] where Root = [32]byte")

		// Check a specific case: are the types being handled the same way?
		if len(fastState.BlockRoots) > 0 && len(ourState.BlockRoots) > 0 {
			// Both should be 8192 elements
			t.Logf("BlockRoots lengths match: %v (fast: %d, our: %d)",
				len(fastState.BlockRoots) == len(ourState.BlockRoots),
				len(fastState.BlockRoots), len(ourState.BlockRoots))
		}
	})
}

func compareHashes(t *testing.T, fieldName string, fastHash, ourHash [32]byte) {
	match := fastHash == ourHash
	status := "✓"
	if !match {
		status = "✗"
	}

	t.Logf("%s %s:", status, fieldName)
	if !match {
		t.Logf("  fast: %s", hex.EncodeToString(fastHash[:]))
		t.Logf("  our:  %s", hex.EncodeToString(ourHash[:]))
	}
}

// Helper functions to hash basic types
func hashUint64(v uint64) [32]byte {
	var buf [32]byte
	buf[0] = byte(v)
	buf[1] = byte(v >> 8)
	buf[2] = byte(v >> 16)
	buf[3] = byte(v >> 24)
	buf[4] = byte(v >> 32)
	buf[5] = byte(v >> 40)
	buf[6] = byte(v >> 48)
	buf[7] = byte(v >> 56)
	return buf
}

func hashBytes32(v [32]byte) [32]byte {
	return v
}
