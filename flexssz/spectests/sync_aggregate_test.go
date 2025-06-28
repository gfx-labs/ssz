package spectests

import (
	"testing"

	"github.com/gfx-labs/ssz/flexssz"
	"github.com/stretchr/testify/require"
)

func TestSyncAggregateWithBitVector(t *testing.T) {
	// SyncAggregate uses a fixed-size bitvector
	syncAgg := &SyncAggregate{
		SyncCommiteeBits:      make([]byte, 64), // 512 bits = 64 bytes
		SyncCommiteeSignature: [96]byte{},
	}

	// Set every other bit
	for i := 0; i < 512; i += 2 {
		err := flexssz.SetBit(syncAgg.SyncCommiteeBits, i)
		require.NoError(t, err)
	}

	// Encode
	encoded, err := flexssz.EncodeStruct(syncAgg)
	require.NoError(t, err)

	// Decode
	var decoded SyncAggregate
	err = flexssz.DecodeStruct(encoded, &decoded)
	require.NoError(t, err)

	// Verify bits
	for i := 0; i < 512; i++ {
		bit, err := flexssz.GetBit(decoded.SyncCommiteeBits, i)
		require.NoError(t, err)
		if i%2 == 0 {
			require.True(t, bit, "Even bit %d should be set", i)
		} else {
			require.False(t, bit, "Odd bit %d should not be set", i)
		}
	}

	// Verify signature
	require.Equal(t, syncAgg.SyncCommiteeSignature, decoded.SyncCommiteeSignature)
}

func TestBeaconStateAltairBitListFields(t *testing.T) {
	// BeaconStateAltair has participation fields that are byte slices
	// These are treated as regular byte slices, not bitlists
	state := &BeaconStateAltair{
		GenesisTime:                1000,
		GenesisValidatorsRoot:      make([]byte, 32),
		Slot:                       100,
		Fork:                       &Fork{
			PreviousVersion: []byte{0, 0, 0, 0},
			CurrentVersion:  []byte{1, 0, 0, 0},
			Epoch:           0,
		},
		LatestBlockHeader:          &BeaconBlockHeader{
			Slot:          0,
			ProposerIndex: 0,
			ParentRoot:    make([]byte, 32),
			StateRoot:     make([]byte, 32),
			BodyRoot:      make([]byte, 32),
		},
		BlockRoots:                 make([][]byte, 8192),
		StateRoots:                 make([][]byte, 8192),
		Eth1Data:                   &Eth1Data{
			DepositRoot:  make([]byte, 32),
			DepositCount: 0,
			BlockHash:    make([]byte, 32),
		},
		Validators:                 []*Validator{},
		Balances:                   []uint64{},
		RandaoMixes:                make([][]byte, 65536),
		Slashings:                  make([]uint64, 8192),
		PreviousEpochParticipation: make([]byte, 1000), // Regular byte slice
		CurrentEpochParticipation:  make([]byte, 1000), // Regular byte slice
		JustificationBits:          []byte{0xFF},       // 1 byte
		PreviousJustifiedCheckpoint: &Checkpoint{
			Epoch: 0,
			Root:  make([]byte, 32),
		},
		CurrentJustifiedCheckpoint:  &Checkpoint{
			Epoch: 0,
			Root:  make([]byte, 32),
		},
		FinalizedCheckpoint:        &Checkpoint{
			Epoch: 0,
			Root:  make([]byte, 32),
		},
		InactivityScores:           []uint64{},
		CurrentSyncCommittee:       &SyncCommittee{
			PubKeys:         make([][]byte, 512),
			AggregatePubKey: [48]byte{},
		},
		NextSyncCommittee:          &SyncCommittee{
			PubKeys:         make([][]byte, 512),
			AggregatePubKey: [48]byte{},
		},
	}

	// Initialize arrays
	for i := 0; i < 8192; i++ {
		state.BlockRoots[i] = make([]byte, 32)
		state.StateRoots[i] = make([]byte, 32)
	}
	for i := 0; i < 65536; i++ {
		state.RandaoMixes[i] = make([]byte, 32)
	}
	for i := 0; i < 512; i++ {
		state.CurrentSyncCommittee.PubKeys[i] = make([]byte, 48)
		state.NextSyncCommittee.PubKeys[i] = make([]byte, 48)
	}

	// Set some participation bits
	for i := 0; i < 100; i++ {
		state.PreviousEpochParticipation[i] = byte(i % 256)
		state.CurrentEpochParticipation[i] = byte((i * 2) % 256)
	}

	// Encode
	encoded, err := flexssz.EncodeStruct(state)
	require.NoError(t, err)
	require.NotEmpty(t, encoded)

	// Decode
	var decoded BeaconStateAltair
	err = flexssz.DecodeStruct(encoded, &decoded)
	require.NoError(t, err)

	// Verify participation fields
	require.Equal(t, state.PreviousEpochParticipation, decoded.PreviousEpochParticipation)
	require.Equal(t, state.CurrentEpochParticipation, decoded.CurrentEpochParticipation)
	require.Equal(t, state.JustificationBits, decoded.JustificationBits)
}