package spectests

import (
	"testing"

	"github.com/gfx-labs/ssz/flexssz"
	"github.com/stretchr/testify/require"
)

func TestAttestationWithBitList(t *testing.T) {
	// Create an attestation with bitlist
	attestation := &Attestation{
		AggregationBits: flexssz.NewBitList(128), // 128 bits
		Data: &AttestationData{
			Slot:  100,
			Index: 1,
			BeaconBlockHash: Hash{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16,
				17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
			Source: &Checkpoint{
				Epoch: 10,
				Root:  []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16,
					17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
			},
			Target: &Checkpoint{
				Epoch: 11,
				Root:  []byte{32, 31, 30, 29, 28, 27, 26, 25, 24, 23, 22, 21, 20, 19, 18, 17,
					16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1},
			},
		},
		Signature: [96]byte{}, // Empty signature for test
	}

	// Set some bits in the aggregation
	require.NoError(t, flexssz.SetBit(attestation.AggregationBits, 0))
	require.NoError(t, flexssz.SetBit(attestation.AggregationBits, 5))
	require.NoError(t, flexssz.SetBit(attestation.AggregationBits, 10))
	require.NoError(t, flexssz.SetBit(attestation.AggregationBits, 127))

	// Encode the attestation
	encoded, err := flexssz.EncodeStruct(attestation)
	require.NoError(t, err)
	require.NotEmpty(t, encoded)

	// Decode into a new attestation
	var decoded Attestation
	err = flexssz.DecodeStruct(encoded, &decoded)
	require.NoError(t, err)

	// Verify the data matches
	require.Equal(t, attestation.Data.Slot, decoded.Data.Slot)
	require.Equal(t, attestation.Data.Index, decoded.Data.Index)
	require.Equal(t, attestation.Data.BeaconBlockHash, decoded.Data.BeaconBlockHash)

	// Check that the bits are preserved
	bit0, err := flexssz.GetBit(decoded.AggregationBits, 0)
	require.NoError(t, err)
	require.True(t, bit0)

	bit5, err := flexssz.GetBit(decoded.AggregationBits, 5)
	require.NoError(t, err)
	require.True(t, bit5)

	bit10, err := flexssz.GetBit(decoded.AggregationBits, 10)
	require.NoError(t, err)
	require.True(t, bit10)

	bit127, err := flexssz.GetBit(decoded.AggregationBits, 127)
	require.NoError(t, err)
	require.True(t, bit127)

	// Check that unset bits remain unset
	bit1, err := flexssz.GetBit(decoded.AggregationBits, 1)
	require.NoError(t, err)
	require.False(t, bit1)
}

func TestPendingAttestationWithBitList(t *testing.T) {
	// Test with PendingAttestation which also uses bitlist
	pending := &PendingAttestation{
		AggregationBits: make([]byte, 64), // 512 bits
		Data: &AttestationData{
			Slot:            200,
			Index:           2,
			BeaconBlockHash: Hash{},
			Source:          &Checkpoint{Epoch: 20, Root: make([]byte, 32)},
			Target:          &Checkpoint{Epoch: 21, Root: make([]byte, 32)},
		},
		InclusionDelay: 1,
		ProposerIndex:  100,
	}

	// Set pattern of bits
	for i := 0; i < 512; i += 10 {
		if err := flexssz.SetBit(pending.AggregationBits, i); err != nil {
			break // Stop if we go beyond the actual bits
		}
	}

	// Encode
	encoded, err := flexssz.EncodeStruct(pending)
	require.NoError(t, err)

	// Decode
	var decoded PendingAttestation
	err = flexssz.DecodeStruct(encoded, &decoded)
	require.NoError(t, err)

	// Verify fields
	require.Equal(t, pending.Data.Slot, decoded.Data.Slot)
	require.Equal(t, pending.InclusionDelay, decoded.InclusionDelay)
	require.Equal(t, pending.ProposerIndex, decoded.ProposerIndex)

	// Check bit pattern
	for i := 0; i < 512; i++ {
		bit, err := flexssz.GetBit(decoded.AggregationBits, i)
		if err != nil {
			break // Beyond actual bits
		}
		if i%10 == 0 {
			require.True(t, bit, "Bit %d should be set", i)
		} else {
			require.False(t, bit, "Bit %d should not be set", i)
		}
	}
}

func TestEmptyBitList(t *testing.T) {
	// Test with empty bitlist
	attestation := &Attestation{
		AggregationBits: []byte{}, // Empty bitlist
		Data: &AttestationData{
			Slot:            300,
			Index:           3,
			BeaconBlockHash: Hash{},
			Source:          &Checkpoint{Epoch: 30, Root: make([]byte, 32)},
			Target:          &Checkpoint{Epoch: 31, Root: make([]byte, 32)},
		},
		Signature: [96]byte{},
	}

	// Encode
	encoded, err := flexssz.EncodeStruct(attestation)
	require.NoError(t, err)

	// Decode
	var decoded Attestation
	err = flexssz.DecodeStruct(encoded, &decoded)
	require.NoError(t, err)

	// Empty bitlist should be preserved
	require.Empty(t, decoded.AggregationBits)
}