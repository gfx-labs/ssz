package main

import (
	"os"
	"io/ioutil"
	"github.com/gfx-labs/ssz/genssz"
)

func main() {
	// Read the schema
	data, err := ioutil.ReadFile("test_list_generation.yaml")
	if err != nil {
		panic(err)
	}
	
	schema, err := genssz.ReadSchemaFromBytes(data)
	if err != nil {
		panic(err)
	}
	
	world, err := genssz.ParseSchemaToWorld(schema)
	if err != nil {
		panic(err)
	}
	
	// Generate code
	f, err := genssz.GenerateCode(world, schema)
	if err != nil {
		panic(err)
	}
	
	// Output the generated code
	err = f.Render(os.Stdout)
	if err != nil {
		panic(err)
	}
}