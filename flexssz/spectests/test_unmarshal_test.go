package spectests

import (
	"testing"
	
	"github.com/gfx-labs/ssz/flexssz"
)

func TestUnmarshal(t *testing.T) {
	// Test with struct
	t.Run("Struct", func(t *testing.T) {
		type SimpleStruct struct {
			A uint64 `json:"a"`
			B uint32 `json:"b"`
		}
		
		original := &SimpleStruct{A: 12345, B: 67890}
		
		// Encode
		encoded, err := flexssz.Marshal(original)
		if err != nil {
			t.Fatalf("Failed to encode: %v", err)
		}
		
		// Decode using new Unmarshal
		decoded := &SimpleStruct{}
		err = flexssz.Unmarshal(encoded, decoded)
		if err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}
		
		if decoded.A != original.A || decoded.B != original.B {
			t.Errorf("Values don't match: got %+v, want %+v", decoded, original)
		}
		
		t.Logf("✓ Struct unmarshal successful")
	})
	
	// Test with slice
	t.Run("Slice", func(t *testing.T) {
		type TestStruct struct {
			Items []uint64 `json:"items" ssz-max:"100"`
		}
		
		original := &TestStruct{Items: []uint64{1, 2, 3, 4, 5}}
		
		// Encode
		encoded, err := flexssz.Marshal(original)
		if err != nil {
			t.Fatalf("Failed to encode: %v", err)
		}
		
		// Decode using new Unmarshal
		decoded := &TestStruct{}
		err = flexssz.Unmarshal(encoded, decoded)
		if err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}
		
		if len(decoded.Items) != len(original.Items) {
			t.Errorf("Length mismatch: got %d, want %d", len(decoded.Items), len(original.Items))
		}
		
		for i, v := range original.Items {
			if decoded.Items[i] != v {
				t.Errorf("Item %d mismatch: got %d, want %d", i, decoded.Items[i], v)
			}
		}
		
		t.Logf("✓ Slice unmarshal successful")
	})
	
	// Test with list of vectors
	t.Run("ListOfVectors", func(t *testing.T) {
		type TestStruct struct {
			Roots [][]byte `json:"roots" ssz-max:"100" ssz-size:"?,32"`
		}
		
		original := &TestStruct{
			Roots: [][]byte{
				make([]byte, 32),
				make([]byte, 32),
			},
		}
		
		// Fill with test data
		for i := range original.Roots[0] {
			original.Roots[0][i] = byte(i)
		}
		for i := range original.Roots[1] {
			original.Roots[1][i] = byte(i + 32)
		}
		
		// Encode
		encoded, err := flexssz.Marshal(original)
		if err != nil {
			t.Fatalf("Failed to encode: %v", err)
		}
		
		// Decode using new Unmarshal
		decoded := &TestStruct{}
		err = flexssz.Unmarshal(encoded, decoded)
		if err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}
		
		if len(decoded.Roots) != len(original.Roots) {
			t.Errorf("Length mismatch: got %d, want %d", len(decoded.Roots), len(original.Roots))
		}
		
		for i, root := range original.Roots {
			if len(decoded.Roots[i]) != len(root) {
				t.Errorf("Root %d length mismatch: got %d, want %d", i, len(decoded.Roots[i]), len(root))
			}
			for j, b := range root {
				if decoded.Roots[i][j] != b {
					t.Errorf("Root %d byte %d mismatch: got %d, want %d", i, j, decoded.Roots[i][j], b)
				}
			}
		}
		
		t.Logf("✓ List of vectors unmarshal successful")
	})
}