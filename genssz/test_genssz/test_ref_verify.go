package main

import (
	"fmt"
	"github.com/gfx-labs/ssz/genssz/test_genssz/testvarcontainer"
)

func main() {
	// Test that the generated variable container code compiles and works
	tx := testvarcontainer.NewTransaction()
	tx.SetNonce(100)
	tx.SetGas_price(1000000000)
	tx.SetGas_limit(21000)
	
	// Test setting variable fields
	tx.SetTo([]byte{1, 2, 3, 4, 5})
	tx.SetData([]byte("test data"))
	
	fmt.Printf("Transaction fields set successfully\n")
	fmt.Printf("Nonce: %d\n", tx.Nonce())
	fmt.Printf("To: %x\n", tx.To())
	fmt.Printf("Data: %s\n", tx.Data())
	fmt.Printf("SSZSize: %d\n", tx.SSZSize())
	
	// Test marshal/unmarshal
	data, err := tx.MarshalSSZ()
	if err != nil {
		panic(err)
	}
	
	tx2 := testvarcontainer.NewTransaction()
	if err := tx2.UnmarshalSSZ(data); err != nil {
		panic(err)
	}
	
	fmt.Printf("\nAfter unmarshal:\n")
	fmt.Printf("Nonce: %d\n", tx2.Nonce())
	fmt.Printf("To: %x\n", tx2.To())
	fmt.Printf("Data: %s\n", tx2.Data())
}