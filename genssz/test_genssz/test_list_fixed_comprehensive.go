package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/gfx-labs/ssz"
)

func main() {
	fmt.Println("=== Testing Vector Implementation ===\n")
	
	// Test 1: Basic uint32 list operations
	fmt.Println("Test 1: Basic uint32 list")
	list1 := ssz.NewVector[uint32](10, 4)
	
	// Add elements using AppendBytes
	for i := uint32(1); i <= 5; i++ {
		data := make([]byte, 4)
		binary.LittleEndian.PutUint32(data, i)
		if err := list1.AppendBytes(data); err != nil {
			fmt.Printf("Error appending: %v\n", err)
		}
	}
	
	fmt.Printf("Length: %d, Limit: %d\n", list1.Len(), list1.Limit())
	
	// Access elements
	for i := 0; i < list1.Len(); i++ {
		bytes := list1.GetBytes(i)
		val := binary.LittleEndian.Uint32(bytes)
		fmt.Printf("Element %d: %d\n", i, val)
	}
	
	// Test 2: Serialization/Deserialization
	fmt.Println("\nTest 2: Serialization/Deserialization")
	serialized, err := list1.MarshalSSZ()
	if err != nil {
		fmt.Printf("Error marshaling: %v\n", err)
		return
	}
	fmt.Printf("Serialized size: %d bytes\n", len(serialized))
	fmt.Printf("Serialized data: %x\n", serialized)
	
	// Deserialize into new list
	list2 := ssz.NewVector[uint32](10, 4)
	if err := list2.UnmarshalSSZ(serialized); err != nil {
		fmt.Printf("Error unmarshaling: %v\n", err)
		return
	}
	
	fmt.Printf("Deserialized length: %d\n", list2.Len())
	allMatch := true
	for i := 0; i < list2.Len(); i++ {
		bytes1 := list1.GetBytes(i)
		bytes2 := list2.GetBytes(i)
		if !bytes.Equal(bytes1, bytes2) {
			allMatch = false
			fmt.Printf("Mismatch at index %d\n", i)
		}
	}
	if allMatch {
		fmt.Println("✓ All elements match after deserialization")
	}
	
	// Test 3: Hashing
	fmt.Println("\nTest 3: Hashing")
	hash1, err := list1.HashTreeRoot()
	if err != nil {
		fmt.Printf("Error hashing: %v\n", err)
		return
	}
	fmt.Printf("Hash of list with %d elements: %x\n", list1.Len(), hash1)
	
	// Empty list hash
	emptyList := ssz.NewVector[uint32](10, 4)
	emptyHash, err := emptyList.HashTreeRoot()
	if err != nil {
		fmt.Printf("Error hashing empty list: %v\n", err)
		return
	}
	fmt.Printf("Hash of empty list: %x\n", emptyHash)
	
	// Test 4: Fixed-size struct elements
	fmt.Println("\nTest 4: Fixed-size struct elements (12 bytes)")
	type FixedStruct struct {
		Field1 uint64
		Field2 uint32
	}
	
	structList := ssz.NewVector[FixedStruct](5, 12)
	
	// Add structs
	for i := 0; i < 3; i++ {
		elem := make([]byte, 12)
		binary.LittleEndian.PutUint64(elem[0:8], uint64(i*100))
		binary.LittleEndian.PutUint32(elem[8:12], uint32(i))
		if err := structList.AppendBytes(elem); err != nil {
			fmt.Printf("Error appending struct: %v\n", err)
		}
	}
	
	fmt.Printf("Struct list length: %d\n", structList.Len())
	
	// Access struct elements
	for i := 0; i < structList.Len(); i++ {
		bytes := structList.GetBytes(i)
		field1 := binary.LittleEndian.Uint64(bytes[0:8])
		field2 := binary.LittleEndian.Uint32(bytes[8:12])
		fmt.Printf("Struct %d: Field1=%d, Field2=%d\n", i, field1, field2)
	}
	
	// Hash struct list
	structHash, err := structList.HashTreeRoot()
	if err != nil {
		fmt.Printf("Error hashing struct list: %v\n", err)
		return
	}
	fmt.Printf("Hash of struct list: %x\n", structHash)
	
	// Test 5: AppendEmpty
	fmt.Println("\nTest 5: AppendEmpty method")
	emptyBytes, err := structList.AppendEmpty()
	if err != nil {
		fmt.Printf("Error appending empty: %v\n", err)
		return
	}
	fmt.Printf("AppendEmpty returned %d bytes\n", len(emptyBytes))
	
	// Modify the empty element
	binary.LittleEndian.PutUint64(emptyBytes[0:8], 999)
	binary.LittleEndian.PutUint32(emptyBytes[8:12], 777)
	
	// Verify modification
	lastIdx := structList.Len() - 1
	bytes := structList.GetBytes(lastIdx)
	field1 := binary.LittleEndian.Uint64(bytes[0:8])
	field2 := binary.LittleEndian.Uint32(bytes[8:12])
	fmt.Printf("Modified empty element: Field1=%d, Field2=%d\n", field1, field2)
	
	// Test 6: Limit enforcement
	fmt.Println("\nTest 6: Limit enforcement")
	smallList := ssz.NewVector[uint32](2, 4)
	
	for i := 0; i < 3; i++ {
		data := make([]byte, 4)
		binary.LittleEndian.PutUint32(data, uint32(i))
		err := smallList.AppendBytes(data)
		if err != nil {
			fmt.Printf("Append %d failed (expected after limit): %v\n", i, err)
		} else {
			fmt.Printf("Append %d succeeded\n", i)
		}
	}
	
	fmt.Println("\n✓ All tests completed!")
}