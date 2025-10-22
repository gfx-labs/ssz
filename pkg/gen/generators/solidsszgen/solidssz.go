package solidsszgen

import (
	"fmt"
	"os"
	"strings"

	"github.com/gfx-labs/cuessz"
)

// Generate generates optimized fixed-size SSZ types from cuessz schema files
// outputPath should be a directory where files will be written as <TypeName>_generated.go
func Generate(outputPath string, inputPaths []string) error {
	if len(inputPaths) == 0 {
		return fmt.Errorf("no input files specified")
	}

	// Read and merge all input schemas
	mergedDefs := make(map[string]cuessz.Def)
	var packageName string

	for _, inputPath := range inputPaths {
		data, err := os.ReadFile(inputPath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", inputPath, err)
		}

		schema, err := ReadSchemaFromBytes(data)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", inputPath, err)
		}

		// Use the first schema's package name
		if packageName == "" && schema.Metadata != nil {
			packageName = sanitizePackageName(schema.Metadata.Namespace)
		}

		// Merge defs from all schemas
		for name, def := range schema.Defs {
			mergedDefs[name] = def
		}
	}

	// Default package name if not specified
	if packageName == "" {
		packageName = "generated"
	}

	// Generate code (returns map of files, one per type)
	files, err := GenerateCode(packageName, mergedDefs)
	if err != nil {
		return fmt.Errorf("failed to generate code: %w", err)
	}

	// Create output directory if it doesn't exist
	err = os.MkdirAll(outputPath, 0755)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write each type to a separate file
	typeCount := 0
	for typeName, file := range files {
		fileName := fmt.Sprintf("%s/%s_generated.go", outputPath, typeName)

		outputFile, err := os.Create(fileName)
		if err != nil {
			return fmt.Errorf("failed to create output file %s: %w", fileName, err)
		}

		err = file.Render(outputFile)
		outputFile.Close()

		if err != nil {
			return fmt.Errorf("failed to write generated code to %s: %w", fileName, err)
		}

		typeCount++
	}

	fmt.Printf("Successfully generated %d types in %s from %v\n", typeCount, outputPath, inputPaths)
	return nil
}

// sanitizePackageName converts a namespace to a valid Go package name
// Examples: "eth2.beacon" -> "beacon", "examples.zoo" -> "zoo"
func sanitizePackageName(namespace string) string {
	if namespace == "" {
		return "generated"
	}

	// Take the last component after the last dot
	parts := strings.Split(namespace, ".")
	pkgName := parts[len(parts)-1]

	// Replace any remaining invalid characters
	pkgName = strings.ReplaceAll(pkgName, "-", "_")
	pkgName = strings.ReplaceAll(pkgName, " ", "_")

	// Convert to lowercase (Go convention)
	pkgName = strings.ToLower(pkgName)

	return pkgName
}
