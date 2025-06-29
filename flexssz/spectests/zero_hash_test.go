package spectests

import (
	"compress/gzip"
	"encoding/hex"
	"io"
	"os"
	"testing"

	"github.com/gfx-labs/ssz/flexssz"
	dynssz "github.com/pk910/dynamic-ssz"
	"github.com/stretchr/testify/require"
)

func mustPrintHex(data [32]byte, err error) string {
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(data[:])
}

func TestBellatrixBeaconStateZeroHash(t *testing.T) {

	fixture, err := os.Open("./_fixtures/zero_beacon_state_bellatrix.ssz.gz")
	require.NoError(t, err)
	defer fixture.Close()

	dec, err := gzip.NewReader(fixture)
	require.NoError(t, err)

	data, err := io.ReadAll(dec)
	require.NoError(t, err)
	fixture.Close()

	state := &BeaconStateBellatrix{}

	err = flexssz.Unmarshal(data, state)

	rootTree, err := flexssz.HashTreeRoot(state)
	require.NoError(t, err)

	dzRoot, err := dynssz.NewDynSsz(nil).HashTreeRoot(state)
	require.NoError(t, err)
	require.Equal(t, hex.EncodeToString(dzRoot[:]), hex.EncodeToString(rootTree[:]))
}
