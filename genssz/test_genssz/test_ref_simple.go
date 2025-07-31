package main

import (
	"fmt"
	"os"
	
	"github.com/gfx-labs/ssz/genssz"
)

func main() {
	// Test simple ref type generation
	schema := &genssz.Schema{
		Package: "testref",
		Structs: []genssz.Struct{
			{
				Name: "FixedData",
				Fields: []genssz.Field{
					{Name: "value", Type: "uint64"},
					{Name: "flag", Type: "bool"},
				},
			},
			{
				Name: "VariableData",
				Fields: []genssz.Field{
					{Name: "count", Type: "uint32"},
					{Name: "items", Type: "bytes", Limit: 1024},
				},
			},
			{
				Name: "ContainerWithRefs",
				Fields: []genssz.Field{
					{Name: "id", Type: "uint64"},
					{Name: "fixed_ref", Type: "ref", Ref: "FixedData"},
					{Name: "variable_ref", Type: "ref", Ref: "VariableData"},
				},
			},
		},
	}
	
	world := &genssz.World{}
	
	f, err := genssz.GenerateCode(world, schema)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	
	// Print the generated code
	fmt.Println(f)
}