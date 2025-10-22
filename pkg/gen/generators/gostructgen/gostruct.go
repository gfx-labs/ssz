package gostructgen

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/gfx-labs/cuessz"
)

// Generate generates Go structs with SSZ annotations from schema files
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

	// Validation is already done by cuessz.ParseJSON, no need to validate again

	// Generate code
	file, err := GenerateCode(packageName, mergedDefs)
	if err != nil {
		return fmt.Errorf("failed to generate code: %w", err)
	}

	// Render to string
	src := file.GoString()

	// Post-process to reorder struct tags (JSON first, then SSZ tags)
	src = reorderStructTags(src)

	// Write to output file
	err = os.WriteFile(outputPath, []byte(src), 0644)
	if err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Printf("Successfully generated %s from %v\n", outputPath, inputPaths)
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

// reorderStructTags reorders struct tags so JSON comes first, then SSZ-related tags.
// Jennifer's Tag() method sorts tags alphabetically, so we need to post-process to fix the order.
// Desired order: json, ssz, ssz-size, ssz-max
func reorderStructTags(src string) string {
	// Regex to match struct tags (backtick-quoted strings)
	tagRegex := regexp.MustCompile("`[^`]+`")

	return tagRegex.ReplaceAllStringFunc(src, func(match string) string {
		// Remove backticks
		tagContent := match[1 : len(match)-1]

		// Parse individual tag:value pairs
		// Regex to match key:"value" pairs
		pairRegex := regexp.MustCompile(`(\w+(?:-\w+)*):"([^"]*)"`)
		matches := pairRegex.FindAllStringSubmatch(tagContent, -1)

		if len(matches) == 0 {
			return match // Return unchanged if no tags found
		}

		// Extract tags into a map
		tagMap := make(map[string]string)
		for _, m := range matches {
			key := m[1]
			value := m[2]
			tagMap[key] = value
		}

		// Build tag string in the desired order
		var parts []string

		// Add JSON tag first
		if jsonValue, ok := tagMap["json"]; ok {
			parts = append(parts, fmt.Sprintf(`json:"%s"`, jsonValue))
		}

		// Add SSZ tags in specific order
		sszTagOrder := []string{"ssz", "ssz-size", "ssz-max", "ssz-index"}
		for _, key := range sszTagOrder {
			if value, ok := tagMap[key]; ok {
				parts = append(parts, fmt.Sprintf(`%s:"%s"`, key, value))
			}
		}

		// Reconstruct tag with backticks
		if len(parts) == 0 {
			return match // Return unchanged if no recognized tags
		}

		return "`" + strings.Join(parts, " ") + "`"
	})
}
