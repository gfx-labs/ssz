package examples

import (
	"testing"

	"github.com/gfx-labs/ssz/pkg/solidssz"
)

// TestReplaceDirective verifies that the local replace directive works
// and we can use the solidssz package from the parent directory
func TestReplaceDirective(t *testing.T) {
	// Test that we can use the local solidssz package through the replace directive
	bl := solidssz.NewBitlist(32)
	if bl == nil {
		t.Fatal("Failed to create bitlist")
	}

	// Test basic operations
	err := bl.Append(true)
	if err != nil {
		t.Fatalf("Failed to append to bitlist: %v", err)
	}

	if bl.Len() != 1 {
		t.Fatalf("Expected length 1, got %d", bl.Len())
	}

	val, err := bl.Get(0)
	if err != nil {
		t.Fatalf("Failed to get bit: %v", err)
	}
	if !val {
		t.Fatal("Expected bit to be true")
	}

	t.Log("✓ Local replace directive working correctly")
}
