package spectests

import (
	"fmt"
	"testing"

	"github.com/gfx-labs/ssz/flexssz"
	dynssz "github.com/pk910/dynamic-ssz"
	"github.com/stretchr/testify/require"
)

// Test struct with just HistoricalRoots field for benchmarking
type HistoricalRootsStruct struct {
	Roots [][32]byte `ssz-max:"16777216"`
}

func generateHistoricalRoots(count int) [][32]byte {
	roots := make([][32]byte, count)
	for i := range roots {
		for j := range roots[i] {
			roots[i][j] = byte((i + j) % 256)
		}
	}
	return roots
}

func BenchmarkHistoricalRoots(b *testing.B) {
	// Test with different sizes
	sizes := []int{0, 10, 100, 1000, 10000}

	for _, size := range sizes {
		roots := generateHistoricalRoots(size)
		testStruct := &HistoricalRootsStruct{Roots: roots}

		b.Run("flexssz/"+fmt.Sprintf("%d_roots", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := flexssz.HashTreeRoot(testStruct)
				if err != nil {
					b.Fatal(err)
				}
			}
		})

		b.Run("dynamic-ssz/"+fmt.Sprintf("%d_roots", size), func(b *testing.B) {
			b.ResetTimer()
			dynSsz := dynssz.NewDynSsz(nil)
			for i := 0; i < b.N; i++ {
				_, err := dynSsz.HashTreeRoot(testStruct)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkComplexStruct(b *testing.B) {
	// Benchmark with a more complex structure similar to BeaconState
	type ComplexStruct struct {
		GenesisTime          uint64     `ssz-size:"8"`
		GenesisValidatorsRoot [32]byte   `ssz-size:"32"`
		Slot                 uint64     `ssz-size:"8"`
		HistoricalRoots      [][32]byte `ssz-max:"16777216"`
		Balances             []uint64   `ssz-max:"1099511627776"`
		RandaoMixes          [][32]byte `ssz-size:"65536,32"`
	}

	// Create test data
	historicalRoots := generateHistoricalRoots(100)
	balances := make([]uint64, 1000)
	for i := range balances {
		balances[i] = uint64(32000000000)
	}
	randaoMixes := make([][32]byte, 65536)
	for i := range randaoMixes {
		for j := range randaoMixes[i] {
			randaoMixes[i][j] = byte((i + j) % 256)
		}
	}

	testStruct := &ComplexStruct{
		GenesisTime:          1234567890,
		GenesisValidatorsRoot: [32]byte{1, 2, 3, 4, 5},
		Slot:                 100000,
		HistoricalRoots:      historicalRoots,
		Balances:             balances,
		RandaoMixes:          randaoMixes,
	}

	b.Run("flexssz", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := flexssz.HashTreeRoot(testStruct)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("dynamic-ssz", func(b *testing.B) {
		b.ResetTimer()
		dynSsz := dynssz.NewDynSsz(nil)
		for i := 0; i < b.N; i++ {
			_, err := dynSsz.HashTreeRoot(testStruct)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkListMerkleization(b *testing.B) {
	// Benchmark specifically list merkleization performance
	sizes := []int{1, 10, 100, 1000, 10000, 100000}

	for _, size := range sizes {
		// Create a simple list of uint64s
		type Uint64List struct {
			Values []uint64 `ssz-max:"1099511627776"`
		}

		values := make([]uint64, size)
		for i := range values {
			values[i] = uint64(i)
		}
		testStruct := &Uint64List{Values: values}

		b.Run("flexssz/uint64_list_"+fmt.Sprintf("%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := flexssz.HashTreeRoot(testStruct)
				if err != nil {
					b.Fatal(err)
				}
			}
		})

		b.Run("dynamic-ssz/uint64_list_"+fmt.Sprintf("%d", size), func(b *testing.B) {
			b.ResetTimer()
			dynSsz := dynssz.NewDynSsz(nil)
			for i := 0; i < b.N; i++ {
				_, err := dynSsz.HashTreeRoot(testStruct)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func TestBenchmarkCorrectness(t *testing.T) {
	// Verify that both libraries produce the same results
	roots := generateHistoricalRoots(100)
	testStruct := &HistoricalRootsStruct{Roots: roots}

	flexRoot, err := flexssz.HashTreeRoot(testStruct)
	require.NoError(t, err)

	dynRoot, err := dynssz.NewDynSsz(nil).HashTreeRoot(testStruct)
	require.NoError(t, err)

	require.Equal(t, flexRoot, dynRoot, "Merkle roots should match between implementations")
}