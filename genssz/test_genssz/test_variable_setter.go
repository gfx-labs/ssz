package main

import (
	"fmt"
	testvarcontainer "github.com/gfx-labs/ssz/genssz/test_genssz/testvarcontainer"
)

func main() {
	// Create a new Transaction
	tx := testvarcontainer.NewTransaction()
	
	// Set fixed fields
	tx.SetNonce(1)
	tx.SetGas_price(1000000000) // 1 gwei
	tx.SetGas_limit(21000)
	tx.SetValue(1000000000000000000) // 1 ETH
	tx.SetV(27)
	tx.SetR(12345)
	tx.SetS(67890)
	
	// Set variable fields using new setters
	to := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 
	             0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14}
	tx.SetTo(to)
	
	data := []byte("hello world")
	tx.SetData(data)
	
	// Print sizes
	fmt.Printf("Transaction SSZSize: %d bytes\n", tx.SSZSize())
	
	// Now update data with a larger value
	newData := []byte("hello world, this is a much longer data field")
	tx.SetData(newData)
	
	fmt.Printf("After update, Transaction SSZSize: %d bytes\n", tx.SSZSize())
	
	// Serialize to verify offsets are correct
	sszData, err := tx.MarshalSSZ()
	if err != nil {
		panic(err)
	}
	
	// Deserialize to verify
	tx2 := testvarcontainer.NewTransaction()
	err = tx2.UnmarshalSSZ(sszData)
	if err != nil {
		panic(err)
	}
	
	fmt.Printf("Deserialized to: %x\n", tx2.To())
	fmt.Printf("Deserialized data: %x\n", tx2.Data())
	fmt.Printf("Deserialized data (string): %s\n", tx2.Data())
}