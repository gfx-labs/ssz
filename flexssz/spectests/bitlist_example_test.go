package spectests

import (
	"fmt"
	"testing"
)

// Example_bitListUsage demonstrates how to use BitList with SSZ structs
func Example_bitListUsage() {
	// Create an attestation with aggregation bits
	attestation := &Attestation{
		AggregationBits: nil, // We'll set this with our BitList
		Data: &AttestationData{
			Slot:  12345,
			Index: 0,
			BeaconBlockHash: Hash{}, // zero hash for example
			Source: &Checkpoint{Epoch: 100, Root: make([]byte, 32)},
			Target: &Checkpoint{Epoch: 101, Root: make([]byte, 32)},
		},
		Signature: [96]byte{}, // zero signature for example
	}

	// Create a BitList for aggregation bits (e.g., 128 validators)
	numValidators := uint64(128)
	aggregationBits := NewBitList(numValidators)

	// Set bits for validators that participated (e.g., validators at indices 0, 5, 10, 15, etc.)
	for i := uint64(0); i < numValidators; i += 5 {
		aggregationBits.SetBit(i)
	}

	// Assign to attestation
	attestation.AggregationBits = aggregationBits.Bytes()

	fmt.Printf("Created attestation with %d aggregation bits\n", aggregationBits.Len())
	fmt.Printf("Number of participating validators: %d\n", countSetBits(aggregationBits))

	// Output:
	// Created attestation with 128 aggregation bits
	// Number of participating validators: 26
}

// Example_bitVectorUsage demonstrates BitVector usage
func Example_bitVectorUsage() {
	// Create a BitVector for a fixed-size bit field (e.g., 256 bits)
	bv := NewBitVector(256)

	// Set every 8th bit
	for i := uint64(0); i < 256; i += 8 {
		bv.SetBit(i)
	}

	fmt.Printf("BitVector has %d bits total\n", bv.Len())
	fmt.Printf("BitVector uses %d bytes\n", len(bv))

	// Output:
	// BitVector has 256 bits total
	// BitVector uses 32 bytes
}

func TestBitListWithSSZStruct(t *testing.T) {
	// This test demonstrates that our BitList implementation
	// is compatible with the SSZ struct tags

	// Create a PendingAttestation with BitList
	pending := &PendingAttestation{
		AggregationBits: nil,
		Data: &AttestationData{
			Slot:  1000,
			Index: 1,
			BeaconBlockHash: Hash{},
			Source: &Checkpoint{Epoch: 10, Root: make([]byte, 32)},
			Target: &Checkpoint{Epoch: 11, Root: make([]byte, 32)},
		},
		InclusionDelay: 1,
		ProposerIndex:  100,
	}

	// Create BitList with 512 validators
	bl := NewBitList(512)
	
	// Simulate 75% participation
	for i := uint64(0); i < 512; i++ {
		if i%4 != 0 { // 3 out of 4 validators participate
			bl.SetBit(i)
		}
	}

	pending.AggregationBits = bl.Bytes()

	// Validate the bitlist
	err := ValidateBitList(bl, 2048) // max is 2048 as per struct tag
	if err != nil {
		t.Fatalf("BitList validation failed: %v", err)
	}

	// Verify we can read it back
	readBL := BitList(pending.AggregationBits)
	if readBL.Len() != 512 {
		t.Errorf("Expected bitlist length 512, got %d", readBL.Len())
	}

	// Count participation
	participantCount := 0
	for i := uint64(0); i < 512; i++ {
		if bit, _ := readBL.GetBit(i); bit {
			participantCount++
		}
	}

	expectedParticipants := 384 // 75% of 512
	if participantCount != expectedParticipants {
		t.Errorf("Expected %d participants, got %d", expectedParticipants, participantCount)
	}
}

// Helper function to count set bits in a BitList
func countSetBits(bl BitList) int {
	count := 0
	for i := uint64(0); i < bl.Len(); i++ {
		if bit, _ := bl.GetBit(i); bit {
			count++
		}
	}
	return count
}