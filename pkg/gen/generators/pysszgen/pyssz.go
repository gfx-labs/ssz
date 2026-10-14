package pysszgen

import (
	"fmt"
	"os"

	"github.com/gfx-labs/cuessz"
)

// Generate generates Python classes for py-ssz from schema files
func Generate(outputPath string, inputPaths []string) error {
	if len(inputPaths) == 0 {
		return fmt.Errorf("no input files specified")
	}

	// Read and merge all input schemas
	mergedDefs := make(map[string]cuessz.Def)

	for _, inputPath := range inputPaths {
		data, err := os.ReadFile(inputPath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", inputPath, err)
		}

		schema, err := ReadSchemaFromBytes(data)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", inputPath, err)
		}

		// Merge defs from all schemas
		for name, def := range schema.Defs {
			mergedDefs[name] = def
		}
	}

	// Generate code
	code, err := GenerateCode(mergedDefs)
	if err != nil {
		return fmt.Errorf("failed to generate code: %w", err)
	}

	// Write to output file
	err = os.WriteFile(outputPath, []byte(code), 0644)
	if err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Printf("Successfully generated %s from %v\n", outputPath, inputPaths)
	return nil
}
