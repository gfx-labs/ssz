package merkle_tree

import (
	"log"
	"testing"

	dynssz "github.com/pk910/dynamic-ssz"
)

func TestBytes(t *testing.T) {

	//bts, err := BytesRoot(make([]byte, 128))
	//require.NoError(t, err)

	var HistoricRoots struct {
		PreviousEpochParticipation []byte `json:"previous_epoch_participation" ssz-max:"1099511627776"`
	}

	x, _ := dynssz.NewDynSsz(nil).HashTreeRoot(HistoricRoots)
	log.Printf("BytesRoot: %x", x)

}
