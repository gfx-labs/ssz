package genssz

import (
	"os"
	"testing"
)

func TestReadSchemaFromBytes(t *testing.T) {
	yamlData := []byte(`
structs:
  - name: TestStruct
    type: container
    children:
      - name: field1
        type: uint8
      - name: field2
        type: list
        limit: 100
`)

	schema, err := ReadSchemaFromBytes(yamlData)
	if err != nil {
		t.Fatalf("ReadSchemaFromBytes failed: %v", err)
	}

	if len(schema.Structs) != 1 {
		t.Errorf("Expected 1 struct, got %d", len(schema.Structs))
	}

	if schema.Structs[0].Name != "TestStruct" {
		t.Errorf("Expected struct name 'TestStruct', got %s", schema.Structs[0].Name)
	}
}

func TestParseSchemaToWorld(t *testing.T) {
	yamlData := []byte(`
structs:
  - name: Container1
    type: container
    children:
      - name: field1
        type: uint32
      - name: field2
        type: bitlist
        limit: 256
      - name: field3
        type: bitvector
        size: 64
      - name: field4
        type: ref
        ref: Container2
  - name: Container2
    type: container
    children:
      - name: data
        type: vector
        size: 32
`)

	schema, err := ReadSchemaFromBytes(yamlData)
	if err != nil {
		t.Fatalf("ReadSchemaFromBytes failed: %v", err)
	}

	world, err := ParseSchemaToWorld(schema)
	if err != nil {
		t.Fatalf("ParseSchemaToWorld failed: %v", err)
	}

	// Check that only top-level types are in the world
	expectedTypes := []string{"Container1", "Container2"}
	for _, typeName := range expectedTypes {
		if _, exists := world.Types[typeName]; !exists {
			t.Errorf("Expected type %s not found in world", typeName)
		}
	}

	// Check that child fields are NOT in the world
	unexpectedTypes := []string{"field1", "field2", "field3", "field4", "data"}
	for _, typeName := range unexpectedTypes {
		if _, exists := world.Types[typeName]; exists {
			t.Errorf("Child type %s should not be in world", typeName)
		}
	}

	// Check that we have exactly 2 types
	if len(world.Types) != 2 {
		t.Errorf("Expected 2 types in world, got %d", len(world.Types))
	}

	// Check container types
	container1 := world.Types["Container1"]
	if container1.Type != "container" {
		t.Errorf("Container1 should be of type 'container', got %s", container1.Type)
	}

	container2 := world.Types["Container2"]
	if container2.Type != "container" {
		t.Errorf("Container2 should be of type 'container', got %s", container2.Type)
	}
}

func TestReadSchemaFromFile(t *testing.T) {
	// Test with the example penguin schema if it exists
	schemaPath := "../examples/penguin/schema.yml"
	if _, err := os.Stat(schemaPath); err == nil {
		data, err := os.ReadFile(schemaPath)
		if err != nil {
			t.Fatalf("Failed to read schema file: %v", err)
		}

		schema, err := ReadSchemaFromBytes(data)
		if err != nil {
			t.Fatalf("ReadSchemaFromBytes failed: %v", err)
		}

		world, err := ParseSchemaToWorld(schema)
		if err != nil {
			t.Fatalf("ParseSchemaToWorld failed: %v", err)
		}

		// Verify penguin schema was parsed correctly
		if _, exists := world.Types["Penguin"]; !exists {
			t.Error("Expected 'Penguin' type not found")
		}
	}
}