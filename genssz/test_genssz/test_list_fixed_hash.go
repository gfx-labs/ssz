package main

import (
	"fmt"
	"github.com/gfx-labs/ssz"
)

func main() {
	// Test hashing of Vector with uint32 elements
	list := ssz.NewVector[uint32](10, 4)
	
	// Add some elements
	data1 := make([]byte, 4)
	data1[0] = 1
	list.AppendBytes(data1)
	
	data2 := make([]byte, 4)
	data2[0] = 2
	list.AppendBytes(data2)
	
	data3 := make([]byte, 4)
	data3[0] = 3
	list.AppendBytes(data3)
	
	// Calculate hash
	hash, err := list.HashTreeRoot()
	if err != nil {
		fmt.Printf("Error calculating hash: %v\n", err)
		return
	}
	
	fmt.Printf("List with %d elements (max %d)\n", list.Len(), list.Limit())
	fmt.Printf("Hash: %x\n", hash)
	
	// Test with fixed-size struct elements
	type FixedItem struct {
		value uint64
		index uint32
	}
	
	list2 := ssz.NewVector[FixedItem](5, 12)
	
	// Add element with value=100, index=1
	elem1 := make([]byte, 12)
	elem1[0] = 100 // value low byte
	elem1[8] = 1   // index low byte
	list2.AppendBytes(elem1)
	
	// Add element with value=200, index=2
	elem2 := make([]byte, 12)
	elem2[0] = 200 // value low byte
	elem2[8] = 2   // index low byte
	list2.AppendBytes(elem2)
	
	hash2, err := list2.HashTreeRoot()
	if err != nil {
		fmt.Printf("Error calculating hash for struct list: %v\n", err)
		return
	}
	
	fmt.Printf("\nStruct list with %d elements (max %d)\n", list2.Len(), list2.Limit())
	fmt.Printf("Hash: %x\n", hash2)
}