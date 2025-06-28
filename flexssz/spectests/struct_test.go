package spectests

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/gfx-labs/ssz/flexssz"
	"github.com/stretchr/testify/require"
)

// Helper to test minimal decoding
func testMinimal(t *testing.T, data []byte) {
	// Try decoding just the fixed fields at the beginning
	type MinimalState struct {
		GenesisTime           uint64 `json:"genesis_time"`
		GenesisValidatorsRoot []byte `json:"genesis_validators_root" ssz-size:"32"`
		Slot                  uint64 `json:"slot"`
	}

	decoder := flexssz.NewDecoder(data)

	// Try to read the first few fields manually
	genesisTime, err := decoder.ReadUint64()
	if err != nil {
		t.Logf("Failed to read genesis time: %v", err)
		return
	}
	t.Logf("Genesis time: %d", genesisTime)

	root, err := decoder.ReadN(32)
	if err != nil {
		t.Logf("Failed to read genesis validators root: %v", err)
		return
	}
	t.Logf("Genesis validators root: %x", root)

	slot, err := decoder.ReadUint64()
	if err != nil {
		t.Logf("Failed to read slot: %v", err)
		return
	}
	t.Logf("Slot: %d", slot)
}

func TestParseBeaconStateBellatrix(t *testing.T) {
	// Read the fixture file
	fixturePath := filepath.Join("_fixtures", "beacon_state_bellatrix.ssz.gz")
	compressedData, err := os.ReadFile(fixturePath)
	if os.IsNotExist(err) {
		t.Skip("Fixture file not found, skipping test")
	}
	require.NoError(t, err, "Failed to read fixture file")

	dataStream, err := gzip.NewReader(bytes.NewBuffer(compressedData))
	require.NoError(t, err, "Failed to decompress fixture file")
	data, err := io.ReadAll(dataStream)
	require.NoError(t, err, "Failed to read fixture file")
	dataStream.Close()

	require.NotEmpty(t, data, "Fixture file is empty")

	// Create an instance of BeaconStateBellatrix to decode into
	var state BeaconStateBellatrix

	// First, validate that the struct has valid SSZ tags
	err = flexssz.PrecacheStructSSZInfo(BeaconStateBellatrix{})
	require.NoError(t, err, "Failed to validate BeaconStateBellatrix struct tags")

	// Decode the SSZ data
	t.Logf("Attempting to decode %d bytes of SSZ data", len(data))
	err = flexssz.DecodeStruct(data, &state)
	if err != nil {
		t.Logf("Decode error: %v", err)
		// Try to understand where it's failing
		// Let's create a minimal test structure
		testMinimal(t, data)
	}
	require.NoError(t, err, "Failed to decode beacon state")

	// Verify some basic properties
	t.Run("basic validation", func(t *testing.T) {
		// Genesis time should be non-zero
		require.NotZero(t, state.GenesisTime, "Genesis time should not be zero")

		// Slot should be non-zero
		require.NotZero(t, state.Slot, "Slot should not be zero")

		// Should have validators
		require.NotEmpty(t, state.Validators, "Should have validators")
		require.NotEmpty(t, state.Balances, "Should have balances")

		// Validators and balances should have same length
		require.Equal(t, len(state.Validators), len(state.Balances),
			"Validators and balances should have same length")

		// Check fixed-size arrays
		require.Len(t, state.BlockRoots, 8192, "BlockRoots should have 8192 entries")
		require.Len(t, state.StateRoots, 8192, "StateRoots should have 8192 entries")
		require.Len(t, state.RandaoMixes, 65536, "RandaoMixes should have 65536 entries")
		require.Len(t, state.Slashings, 8192, "Slashings should have 8192 entries")

		// Each block root should be 32 bytes
		for i, root := range state.BlockRoots {
			if root != nil { // Some might be nil
				require.Len(t, root, 32, "Block root at index %d should be 32 bytes", i)
			}
		}

		// Check required fields are not nil
		require.NotNil(t, state.Fork, "Fork should not be nil")
		require.NotNil(t, state.LatestBlockHeader, "LatestBlockHeader should not be nil")
		require.NotNil(t, state.Eth1Data, "Eth1Data should not be nil")
		require.NotNil(t, state.PreviousJustifiedCheckpoint, "PreviousJustifiedCheckpoint should not be nil")
		require.NotNil(t, state.CurrentJustifiedCheckpoint, "CurrentJustifiedCheckpoint should not be nil")
		require.NotNil(t, state.FinalizedCheckpoint, "FinalizedCheckpoint should not be nil")
		require.NotNil(t, state.CurrentSyncCommittee, "CurrentSyncCommittee should not be nil")
		require.NotNil(t, state.NextSyncCommittee, "NextSyncCommittee should not be nil")
		require.NotNil(t, state.LatestExecutionPayloadHeader, "LatestExecutionPayloadHeader should not be nil")
	})

	t.Run("fork validation", func(t *testing.T) {
		require.Len(t, state.Fork.PreviousVersion, 4, "Previous version should be 4 bytes")
		require.Len(t, state.Fork.CurrentVersion, 4, "Current version should be 4 bytes")
	})

	t.Run("validator validation", func(t *testing.T) {
		// Check first validator if present
		if len(state.Validators) > 0 {
			validator := state.Validators[0]
			require.NotNil(t, validator, "First validator should not be nil")
			require.Len(t, validator.Pubkey, 48, "Validator pubkey should be 48 bytes")
			require.Len(t, validator.WithdrawalCredentials, 32, "Withdrawal credentials should be 32 bytes")
		}
	})

	t.Run("sync committee validation", func(t *testing.T) {
		// Current sync committee
		require.Len(t, state.CurrentSyncCommittee.PubKeys, 512, "Should have 512 sync committee members")
		for i, pubkey := range state.CurrentSyncCommittee.PubKeys {
			require.Len(t, pubkey, 48, "Sync committee pubkey %d should be 48 bytes", i)
		}
		require.Len(t, state.CurrentSyncCommittee.AggregatePubKey, 48, "Aggregate pubkey should be 48 bytes")

		// Next sync committee
		require.Len(t, state.NextSyncCommittee.PubKeys, 512, "Should have 512 next sync committee members")
	})

	t.Run("execution payload header validation", func(t *testing.T) {
		header := state.LatestExecutionPayloadHeader
		require.Len(t, header.ParentHash, 32, "Parent hash should be 32 bytes")
		require.Len(t, header.FeeRecipient, 20, "Fee recipient should be 20 bytes")
		require.Len(t, header.StateRoot, 32, "State root should be 32 bytes")
		require.Len(t, header.ReceiptsRoot, 32, "Receipts root should be 32 bytes")
		require.Len(t, header.LogsBloom, 256, "Logs bloom should be 256 bytes")
		require.Len(t, header.PrevRandao, 32, "Prev randao should be 32 bytes")
		require.Len(t, header.BaseFeePerGas, 32, "Base fee per gas should be 32 bytes")
		require.Len(t, header.BlockHash, 32, "Block hash should be 32 bytes")
		require.Len(t, header.TransactionsRoot, 32, "Transactions root should be 32 bytes")
	})

	// Test round-trip encoding
	t.Run("round-trip", func(t *testing.T) {
		// Encode the decoded state
		encoded, err := flexssz.EncodeStruct(state)
		require.NoError(t, err, "Failed to encode beacon state")

		// The encoded data should match the original
		require.Equal(t, len(data), len(encoded), "Encoded length should match original")

		// For debugging: show size
		t.Logf("Beacon state size: %d bytes", len(data))
		t.Logf("Number of validators: %d", len(state.Validators))
		t.Logf("Slot: %d", state.Slot)
		t.Logf("Genesis time: %d", state.GenesisTime)

		// Decode again to verify round-trip
		var state2 BeaconStateBellatrix
		err = flexssz.DecodeStruct(encoded, &state2)
		require.NoError(t, err, "Failed to decode re-encoded beacon state")

		// Compare key fields
		require.Equal(t, state.GenesisTime, state2.GenesisTime, "Genesis time should match")
		require.Equal(t, state.Slot, state2.Slot, "Slot should match")
		require.Equal(t, len(state.Validators), len(state2.Validators), "Validator count should match")
	})

	// Test exact byte-for-byte reproduction
	t.Run("exact-bytes-reproduction", func(t *testing.T) {
		// Encode the decoded state
		encoded, err := flexssz.EncodeStruct(state)
		require.NoError(t, err, "Failed to encode beacon state")

		// The encoded bytes should be exactly the same as the original
		require.Equal(t, data, encoded, "Re-encoded bytes should exactly match original SSZ data")
	})
}

// Benchmark decoding performance
func BenchmarkDecodeBeaconStateBellatrix(b *testing.B) {
	// Read the fixture file once
	fixturePath := filepath.Join("_fixtures", "beacon_state_bellatrix.ssz")
	data, err := os.ReadFile(fixturePath)
	require.NoError(b, err)

	// Pre-cache struct info
	err = flexssz.PrecacheStructSSZInfo(BeaconStateBellatrix{})
	require.NoError(b, err)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var state BeaconStateBellatrix
		err := flexssz.DecodeStruct(data, &state)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.SetBytes(int64(len(data)))
}

// Benchmark encoding performance
func BenchmarkEncodeBeaconStateBellatrix(b *testing.B) {
	// Read and decode once to get a state
	fixturePath := filepath.Join("_fixtures", "beacon_state_bellatrix.ssz")
	data, err := os.ReadFile(fixturePath)
	require.NoError(b, err)

	var state BeaconStateBellatrix
	err = flexssz.DecodeStruct(data, &state)
	require.NoError(b, err)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		encoded, err := flexssz.EncodeStruct(state)
		if err != nil {
			b.Fatal(err)
		}
		b.SetBytes(int64(len(encoded)))
	}
}
