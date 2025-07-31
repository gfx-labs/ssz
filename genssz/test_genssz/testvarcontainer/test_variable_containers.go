package testvarcontainer

import (
	"fmt"
)

func TestVariableContainers() {
	// Test Transaction
	tx := NewTransaction()
	tx.SetNonce(1)
	tx.SetGas_price(1000000000) // 1 gwei
	tx.SetGas_limit(21000)
	tx.SetValue(1000000000000000000) // 1 ETH
	tx.SetV(27)
	tx.SetR(12345)
	tx.SetS(67890)
	
	// Set variable fields
	tx.to = []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14}
	tx.data = []byte("hello world")
	
	// Serialize
	data, err := tx.MarshalSSZ()
	if err != nil {
		panic(err)
	}
	
	fmt.Printf("Transaction serialized size: %d bytes\n", len(data))
	fmt.Printf("Transaction SSZSize: %d bytes\n", tx.SSZSize())
	
	// Deserialize
	tx2 := NewTransaction()
	err = tx2.UnmarshalSSZ(data)
	if err != nil {
		panic(err)
	}
	
	fmt.Printf("Deserialized nonce: %d\n", tx2.Nonce())
	fmt.Printf("Deserialized gas_price: %d\n", tx2.Gas_price())
	fmt.Printf("Deserialized to: %x\n", tx2.to)
	fmt.Printf("Deserialized data: %s\n", tx2.data)
	
	// Test Block
	block := NewBlock()
	block.SetSlot(12345)
	block.SetProposer_index(42)
	
	// Set attestations
	block.attestations = []uint64{1, 2, 3, 4, 5}
	
	// Note: transactions list of lists is not fully implemented yet
	// block.Transactions = [][]byte{[]byte("tx1"), []byte("tx2")}
	
	blockData, err := block.MarshalSSZ()
	if err != nil {
		panic(err)
	}
	
	fmt.Printf("\nBlock serialized size: %d bytes\n", len(blockData))
	fmt.Printf("Block SSZSize: %d bytes\n", block.SSZSize())
}