package pysszgen

import (
	"encoding/json"
	"fmt"

	"github.com/gfx-labs/cuessz"
	"sigs.k8s.io/yaml"
)

// ReadSchemaFromBytes reads a schema from JSON or YAML bytes
// Converts YAML to JSON, then parses with cuessz for validation
func ReadSchemaFromBytes(data []byte) (*cuessz.Schema, error) {
	// Parse YAML (which also handles JSON since all JSON is valid YAML)
	var yamlData interface{}
	err := yaml.Unmarshal(data, &yamlData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML/JSON: %w", err)
	}

	// Convert to JSON for cuessz
	jsonData, err := json.Marshal(yamlData)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to JSON: %w", err)
	}

	// Parse and validate with cuessz
	schema, err := cuessz.ParseJSON(jsonData)
	if err != nil {
		return nil, fmt.Errorf("schema validation failed: %w", err)
	}

	return schema, nil
}
