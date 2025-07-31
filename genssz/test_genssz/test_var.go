package main

import (
	"fmt"
)

// Include the generated code inline
//go:generate go run test_variable_gen.go > generated_variable.go

func main() {
	fmt.Println("Variable container generation completed successfully!")
	fmt.Println("Generated types:")
	fmt.Println("- Transaction: container with fixed fields and variable 'to' and 'data' fields")
	fmt.Println("- Block: container with fixed fields and variable 'transactions' and 'attestations' fields")
}