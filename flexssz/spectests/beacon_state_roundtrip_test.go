package spectests

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"testing"
	
	"github.com/gfx-labs/ssz/flexssz"
)

func TestBeaconStateBellatrixRoundtrip(t *testing.T) {
	// Read the fixture file
	fixturePath := "_fixtures/beacon_state_bellatrix.ssz.gz"
	
	// Open the gzipped file
	file, err := os.Open(fixturePath)
	if err != nil {
		t.Fatalf("Failed to open fixture file: %v", err)
	}
	defer file.Close()
	
	// Create gzip reader
	gzReader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gzReader.Close()
	
	// Read the original SSZ data
	originalData, err := io.ReadAll(gzReader)
	if err != nil {
		t.Fatalf("Failed to read fixture data: %v", err)
	}
	
	t.Logf("Original data size: %d bytes", len(originalData))
	
	// Create a BeaconStateBellatrix instance
	state := &BeaconStateBellatrix{}
	
	// Unmarshal the data using flexssz
	if err := flexssz.DecodeStruct(originalData, state); err != nil {
		t.Fatalf("Failed to unmarshal SSZ data: %v", err)
	}
	
	// Log some basic info about the unmarshaled state
	t.Logf("Unmarshaled state:")
	t.Logf("  Slot: %d", state.Slot)
	t.Logf("  Number of validators: %d", len(state.Validators))
	t.Logf("  Number of balances: %d", len(state.Balances))
	
	// Marshal the state back to SSZ using flexssz
	marshaledData, err := flexssz.EncodeStruct(state)
	if err != nil {
		t.Fatalf("Failed to marshal SSZ data: %v", err)
	}
	
	t.Logf("Marshaled data size: %d bytes", len(marshaledData))
	
	// Compare the sizes first
	if len(originalData) != len(marshaledData) {
		t.Errorf("Data size mismatch: original=%d, marshaled=%d", len(originalData), len(marshaledData))
		
		// Find where they start to differ
		minLen := len(originalData)
		if len(marshaledData) < minLen {
			minLen = len(marshaledData)
		}
		
		for i := 0; i < minLen; i++ {
			if originalData[i] != marshaledData[i] {
				t.Logf("First difference at byte %d: original=0x%02x, marshaled=0x%02x", i, originalData[i], marshaledData[i])
				
				// Show some context
				start := i - 10
				if start < 0 {
					start = 0
				}
				end := i + 10
				if end > minLen {
					end = minLen
				}
				
				t.Logf("Context (original): %x", originalData[start:end])
				t.Logf("Context (marshaled): %x", marshaledData[start:end])
				break
			}
		}
	}
	
	// Compare the actual bytes
	if !bytes.Equal(originalData, marshaledData) {
		t.Error("Marshaled data does not match original data")
		
		// Additional debugging: check specific offsets that might be problematic
		t.Run("DebugOffsets", func(t *testing.T) {
			// The SSZ format has specific offsets for variable-length fields
			// Let's check the first few offsets
			if len(originalData) >= 4 && len(marshaledData) >= 4 {
				origOffset := uint32(originalData[0]) | uint32(originalData[1])<<8 | uint32(originalData[2])<<16 | uint32(originalData[3])<<24
				marshalOffset := uint32(marshaledData[0]) | uint32(marshaledData[1])<<8 | uint32(marshaledData[2])<<16 | uint32(marshaledData[3])<<24
				t.Logf("First offset - original: %d, marshaled: %d", origOffset, marshalOffset)
			}
		})
	} else {
		t.Log("✓ Round-trip successful: marshaled data matches original exactly!")
	}
	
	// Also test that we can unmarshal the marshaled data successfully
	t.Run("UnmarshalMarshaled", func(t *testing.T) {
		state2 := &BeaconStateBellatrix{}
		if err := flexssz.DecodeStruct(marshaledData, state2); err != nil {
			t.Fatalf("Failed to unmarshal marshaled data: %v", err)
		}
		
		// Quick sanity check
		if state2.Slot != state.Slot {
			t.Errorf("Slot mismatch after second unmarshal: %d vs %d", state2.Slot, state.Slot)
		}
		if len(state2.Validators) != len(state.Validators) {
			t.Errorf("Validator count mismatch after second unmarshal: %d vs %d", len(state2.Validators), len(state.Validators))
		}
	})
	
	// Test hash consistency
	t.Run("HashConsistency", func(t *testing.T) {
		// Calculate hash of original unmarshaled state
		hash1, err := flexssz.HashTreeRootStruct(state)
		if err != nil {
			t.Fatalf("Failed to calculate hash of original state: %v", err)
		}
		
		// Unmarshal and hash again
		state3 := &BeaconStateBellatrix{}
		if err := flexssz.DecodeStruct(marshaledData, state3); err != nil {
			t.Fatalf("Failed to unmarshal for hash test: %v", err)
		}
		
		hash2, err := flexssz.HashTreeRootStruct(state3)
		if err != nil {
			t.Fatalf("Failed to calculate hash of remarshaled state: %v", err)
		}
		
		if hash1 != hash2 {
			t.Errorf("Hash mismatch: original=0x%x, remarshaled=0x%x", hash1, hash2)
		} else {
			t.Logf("✓ Hash consistency maintained: 0x%x", hash1)
		}
	})
}