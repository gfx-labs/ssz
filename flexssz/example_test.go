package flexssz_test

import (
	"fmt"
	"log"

	"github.com/gfx-labs/ssz/flexssz"
	"github.com/holiman/uint256"
)

// Example struct with SSZ tags
type MyBlock struct {
	Slot          uint64       `ssz:"uint64"`
	ProposerIndex uint64       `ssz:"uint64"`
	ParentRoot    [32]byte     `ssz:"vector"`      // Fixed-size array uses 'vector'
	StateRoot     [32]byte     `ssz:"vector"`      // Fixed-size array uses 'vector'
	Signature     []byte       `ssz-max:"96"`      // Dynamic slice uses ssz-max
	Balance       uint256.Int  `ssz:"uint256"`
	SmallBalance  uint256.Int  `ssz:"uint128"`
	Validators    []uint64     `ssz-max:"100"`     // Dynamic slice needs ssz-max
}

func init() {
	// Precache struct SSZ info to validate tags at initialization
	// This will panic if any tags are invalid
	if err := flexssz.PrecacheStructSSZInfo(MyBlock{}); err != nil {
		log.Fatalf("Invalid SSZ tags in MyBlock: %v", err)
	}
}

func ExamplePrecacheStructSSZInfo() {
	// The struct is already validated in init()
	// Now we can safely encode it
	
	block := MyBlock{
		Slot:          12345,
		ProposerIndex: 67,
		Signature:     make([]byte, 96),
		Validators:    []uint64{1, 2, 3, 4, 5},
	}
	block.Balance.SetUint64(1000000)
	block.SmallBalance.SetUint64(50000)
	
	encoded, err := flexssz.EncodeStruct(block)
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("Encoded %d bytes\n", len(encoded))
	// Output: Encoded 272 bytes
}

func ExamplePrecacheStructSSZInfo_validation() {
	// This example shows validation error at parse time
	type InvalidStruct struct {
		// This will fail validation - uint32 tag on uint64 type
		Value uint64 `ssz:"uint32"`
	}
	
	err := flexssz.PrecacheStructSSZInfo(InvalidStruct{})
	if err != nil {
		fmt.Println("Validation error:", err)
	}
	// Output: Validation error: field Value: ssz tag 'uint32' requires Go type uint32, got uint64
}