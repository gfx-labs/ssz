package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/gfx-labs/ssz/genssz"
)

func main() {
	var (
		output = flag.String("output", "", "Output Go file")
	)
	flag.Parse()

	// Get input files from remaining args
	inputFiles := flag.Args()
	
	if len(inputFiles) == 0 || *output == "" {
		fmt.Fprintf(os.Stderr, "Usage: genssz -output generated.go schema1.yml schema2.yml ...\n")
		os.Exit(1)
	}

	// Combine schemas from all input files
	combinedSchema, err := combineSchemas(inputFiles)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to combine schemas: %v\n", err)
		os.Exit(1)
	}

	// Create world
	world, err := genssz.ParseSchemaToWorld(combinedSchema)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create world: %v\n", err)
		os.Exit(1)
	}

	// Generate code
	code, err := genssz.GenerateCode(world, combinedSchema)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate code: %v\n", err)
		os.Exit(1)
	}

	// Write output
	file, err := os.Create(*output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create output file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	if err := code.Render(file); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write output: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully generated %s from %s\n", *output, strings.Join(inputFiles, ", "))
}

// combineSchemas reads multiple schema files and combines them into one
func combineSchemas(files []string) (*genssz.Schema, error) {
	var combinedSchema *genssz.Schema
	seenPackage := false
	
	for _, file := range files {
		// Read schema file
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", file, err)
		}

		// Parse schema
		schema, err := genssz.ReadSchemaFromBytes(data)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", file, err)
		}

		// Handle package
		if schema.Package != "" {
			if seenPackage && combinedSchema.Package != schema.Package {
				return nil, fmt.Errorf("conflicting package names: %s vs %s", combinedSchema.Package, schema.Package)
			}
			if !seenPackage {
				seenPackage = true
				if combinedSchema == nil {
					combinedSchema = &genssz.Schema{Package: schema.Package}
				} else {
					combinedSchema.Package = schema.Package
				}
			}
		}
		
		// Initialize combined schema if needed
		if combinedSchema == nil {
			combinedSchema = &genssz.Schema{}
		}
		
		// Append structs
		combinedSchema.Structs = append(combinedSchema.Structs, schema.Structs...)
	}
	
	if combinedSchema == nil {
		return nil, fmt.Errorf("no schemas found")
	}
	
	if combinedSchema.Package == "" {
		return nil, fmt.Errorf("no package name specified in any schema")
	}
	
	return combinedSchema, nil
}