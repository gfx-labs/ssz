package spectests

import (
	"compress/gzip"
	"encoding/hex"
	"io"
	"os"
	"testing"

	"github.com/ferranbt/fastssz/spectests"
	"github.com/gfx-labs/ssz/flexssz"
	"github.com/stretchr/testify/require"
)

func TestMerkleRootComparison(t *testing.T) {
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

	t.Run("BeaconStateBellatrix", func(t *testing.T) {
		// Unmarshal with both implementations
		fastState := &spectests.BeaconStateBellatrix{}
		if err := fastState.UnmarshalSSZ(originalData); err != nil {
			t.Fatalf("Failed to unmarshal with fastssz: %v", err)
		}

		ourState := &BeaconStateBellatrix{}
		if err := flexssz.Unmarshal(originalData, ourState); err != nil {
			t.Fatalf("Failed to unmarshal with flexssz: %v", err)
		}

		// Compare merkle roots
		fastRoot, err := fastState.HashTreeRoot()
		if err != nil {
			t.Fatalf("Failed to get hash tree root with fastssz: %v", err)
		}

		ourRoot, err := flexssz.HashTreeRoot(ourState)
		if err != nil {
			t.Fatalf("Failed to get hash tree root with flexssz: %v", err)
		}

		t.Logf("fastssz merkle root: 0x%s", hex.EncodeToString(fastRoot[:]))
		t.Logf("flexssz merkle root: 0x%s", hex.EncodeToString(ourRoot[:]))
		require.EqualValues(t, fastRoot[:], ourRoot[:])

	})

}
