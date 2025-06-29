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

func TestExecutionData(t *testing.T) {

	fixture, err := os.Open("./_fixtures/zero_execution_payload_header.ssz.gz")
	require.NoError(t, err)
	defer fixture.Close()

	dec, err := gzip.NewReader(fixture)
	require.NoError(t, err)

	data, err := io.ReadAll(dec)
	require.NoError(t, err)
	fixture.Close()

	state := &ExecutionPayloadHeader{}

	err = flexssz.Unmarshal(data, state)

	rootTree, err := flexssz.HashTreeRoot(state)
	require.NoError(t, err)
	dzRoot, err := dynssz.NewDynSsz(nil).HashTreeRoot(state)

	require.NoError(t, err)
	require.Equal(t, hex.EncodeToString(dzRoot[:]), hex.EncodeToString(rootTree[:]))
}
