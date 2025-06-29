package flexssz

import (
	"testing"

	"github.com/holiman/uint256"
)

func TestHashTreeRoot(t *testing.T) {
	// Define a test struct
	type TestStruct struct {
		Field1 uint64          `ssz:"uint64"`
		Field2 bool            `ssz:"bool"`
		Field3 [32]byte        `ssz-size:"32"`
		Field4 []byte          `ssz-max:"256"`
		Field5 string          `ssz:"string"`
		Field6 []uint32        `ssz-max:"10"`
		Field7 uint256.Int     `ssz:"uint256"`
		Field8 *uint256.Int    `ssz:"uint256"`
	}

	// Create test data
	uint256Val := uint256.NewInt(12345)
	test := &TestStruct{
		Field1: 42,
		Field2: true,
		Field3: [32]byte{1, 2, 3, 4, 5},
		Field4: []byte{6, 7, 8, 9, 10},
		Field5: "hello world",
		Field6: []uint32{11, 12, 13},
		Field7: *uint256Val,
		Field8: uint256Val,
	}

	// Calculate hash tree root
	root, err := HashTreeRoot(test)
	if err != nil {
		t.Fatalf("Failed to calculate hash tree root: %v", err)
	}

	// The root should be non-zero
	var zeroHash [32]byte
	if root == zeroHash {
		t.Error("Hash tree root should not be zero")
	}

	t.Logf("Hash tree root: %x", root)
}

func TestHashTreeRootNestedStruct(t *testing.T) {
	// Define nested structs
	type Inner struct {
		A uint32 `ssz:"uint32"`
		B uint64 `ssz:"uint64"`
	}

	type Outer struct {
		X     uint16  `ssz:"uint16"`
		Y     Inner   `ssz:"container"`
		Z     []byte  `ssz-max:"32"`
		Array [4]uint32 `ssz-size:"4"`
	}

	// Create test data
	test := &Outer{
		X: 100,
		Y: Inner{
			A: 200,
			B: 300,
		},
		Z: []byte{1, 2, 3, 4},
		Array: [4]uint32{10, 20, 30, 40},
	}

	// Calculate hash tree root
	root, err := HashTreeRoot(test)
	if err != nil {
		t.Fatalf("Failed to calculate hash tree root: %v", err)
	}

	// The root should be non-zero
	var zeroHash [32]byte
	if root == zeroHash {
		t.Error("Hash tree root should not be zero")
	}

	t.Logf("Nested struct hash tree root: %x", root)
}

func TestHashTreeRootBitfields(t *testing.T) {
	// Test struct with bitfields
	type BitfieldStruct struct {
		BitVector []byte `ssz:"bitvector" ssz-size:"64"`
		BitList   []byte `ssz:"bitlist" ssz-max:"256"`
	}

	test := &BitfieldStruct{
		BitVector: []byte{0xFF, 0x00, 0xFF, 0x00, 0xFF, 0x00, 0xFF, 0x00}, // 64 bits
		BitList:   []byte{0xFF, 0x00, 0xFF, 0x80}, // Last byte has sentinel bit
	}

	// Calculate hash tree root
	root, err := HashTreeRoot(test)
	if err != nil {
		t.Fatalf("Failed to calculate hash tree root: %v", err)
	}

	// The root should be non-zero
	var zeroHash [32]byte
	if root == zeroHash {
		t.Error("Hash tree root should not be zero")
	}

	t.Logf("Bitfield struct hash tree root: %x", root)
}