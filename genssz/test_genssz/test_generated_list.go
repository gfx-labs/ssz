package main

import (
	"fmt"
	testlist "github.com/gfx-labs/ssz/genssz/test_genssz/testlist"
)

func main() {
	// Test MyUint32List
	list := testlist.NewMyUint32List()
	
	// Add some elements
	for i := uint32(1); i <= 5; i++ {
		err := list.Append(i * 10)
		if err != nil {
			panic(err)
		}
	}
	
	fmt.Printf("List length: %d\n", list.Len())
	fmt.Printf("List limit: %d\n", list.Limit())
	
	// Get elements
	for i := 0; i < list.Len(); i++ {
		fmt.Printf("Element %d: %d\n", i, list.Get(i))
	}
	
	// Test SSZ serialization
	sszData, err := list.MarshalSSZ()
	if err != nil {
		panic(err)
	}
	fmt.Printf("SSZ size: %d bytes\n", len(sszData))
	
	// Test deserialization
	list2 := testlist.NewMyUint32List()
	err = list2.UnmarshalSSZ(sszData)
	if err != nil {
		panic(err)
	}
	
	fmt.Printf("Deserialized list length: %d\n", list2.Len())
	
	// Test MyFixedItemList
	fixedList := testlist.NewMyFixedItemList()
	
	item1 := testlist.NewFixedItemWithValues(100, 1)
	item2 := testlist.NewFixedItemWithValues(200, 2)
	
	fixedList.Append(item1)
	fixedList.Append(item2)
	
	fmt.Printf("\nFixed item list length: %d\n", fixedList.Len())
	
	// Test SSZ for fixed item list
	fixedSSZ, err := fixedList.MarshalSSZ()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Fixed list SSZ size: %d bytes\n", len(fixedSSZ))
	
	// Test hash tree root
	hash, err := list.HashTreeRoot()
	if err != nil {
		panic(err)
	}
	fmt.Printf("\nList hash tree root: %x\n", hash)
}