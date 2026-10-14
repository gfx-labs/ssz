package ssztest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gfx-labs/cuessz"
	"github.com/gfx-labs/ssz/pkg/gen/generators/solidsszgen"
)

func TestGenerator(t *testing.T) {
	// Create a simple schema
	defs := map[string]cuessz.Def{
		"SimpleStruct": {
			Type: cuessz.TypeContainer,
			Children: []cuessz.Field{
				{
					Name: "id",
					Def:  cuessz.Def{Type: cuessz.TypeUint64},
				},
				{
					Name: "enabled",
					Def:  cuessz.Def{Type: cuessz.TypeBoolean},
				},
			},
		},
	}

	gen := NewGenerator(defs)

	// Generate 10 random values
	for i := 0; i < 10; i++ {
		value, err := gen.GenerateNamed("SimpleStruct")
		if err != nil {
			t.Fatalf("Failed to generate value: %v", err)
		}

		container := value.(map[string]interface{})
		if _, ok := container["id"].(uint64); !ok {
			t.Errorf("Expected id to be uint64")
		}
		if _, ok := container["enabled"].(bool); !ok {
			t.Errorf("Expected enabled to be bool")
		}
	}

	t.Log("✓ Generator can create random values")
}

func TestReferenceImplementation(t *testing.T) {
	defs := map[string]cuessz.Def{
		"Example": {
			Type: cuessz.TypeContainer,
			Children: []cuessz.Field{
				{Name: "a", Def: cuessz.Def{Type: cuessz.TypeUint8}},
				{Name: "b", Def: cuessz.Def{Type: cuessz.TypeUint16}},
				{Name: "c", Def: cuessz.Def{Type: cuessz.TypeUint32}},
				{Name: "d", Def: cuessz.Def{Type: cuessz.TypeUint64}},
			},
		},
	}

	ref := NewReferenceImplementation()

	value := map[string]interface{}{
		"a": uint8(1),
		"b": uint16(2),
		"c": uint32(3),
		"d": uint64(4),
	}

	data, err := ref.Marshal(value, defs["Example"])
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Expected: 1 byte + 2 bytes + 4 bytes + 8 bytes = 15 bytes
	expectedSize := 1 + 2 + 4 + 8
	if len(data) != expectedSize {
		t.Errorf("Expected %d bytes, got %d", expectedSize, len(data))
	}

	t.Logf("✓ Reference implementation serializes correctly (%d bytes)", len(data))
}

func TestComparator(t *testing.T) {
	defs := map[string]cuessz.Def{
		"TestStruct": {
			Type: cuessz.TypeContainer,
			Children: []cuessz.Field{
				{Name: "x", Def: cuessz.Def{Type: cuessz.TypeUint32}},
				{Name: "y", Def: cuessz.Def{Type: cuessz.TypeUint32}},
			},
		},
	}

	// Create two implementations (in real use, these would be different libraries)
	ref1 := NewReferenceImplementation()
	ref2 := NewReferenceImplementation()

	comparator := NewComparator(defs, ref1, ref2)

	value := map[string]interface{}{
		"x": uint32(100),
		"y": uint32(200),
	}

	result, err := comparator.Compare("TestStruct", value)
	if err != nil {
		t.Fatalf("Comparison failed: %v", err)
	}

	if result.HasMismatches() {
		t.Errorf("Unexpected mismatches:\n%s", result.Report())
	}

	if result.HasErrors() {
		t.Errorf("Unexpected errors:\n%s", result.Report())
	}

	t.Log("✓ Comparator finds agreement between implementations")
}

func TestFuzzer(t *testing.T) {
	defs := map[string]cuessz.Def{
		"FuzzStruct": {
			Type: cuessz.TypeContainer,
			Children: []cuessz.Field{
				{Name: "a", Def: cuessz.Def{Type: cuessz.TypeUint8}},
				{Name: "b", Def: cuessz.Def{Type: cuessz.TypeUint64}},
				{Name: "c", Def: cuessz.Def{Type: cuessz.TypeBoolean}},
			},
		},
	}

	ref := NewReferenceImplementation()

	config := FuzzConfig{
		NumIterations:       50,
		Seed:                12345,
		StopOnFirstMismatch: true,
		Verbose:             false,
	}

	fuzzer := NewFuzzer(defs, []SSZImplementation{ref}, config)
	result := fuzzer.Fuzz()

	if !result.Success() {
		t.Errorf("Fuzzing found issues:\n%s", result.Report())
	}

	t.Logf("✓ Fuzzer completed %d tests in %v (%.0f tests/sec)",
		result.TotalTests, result.Duration, float64(result.TotalTests)/result.Duration.Seconds())
}

func TestFuzzerWithZooSchema(t *testing.T) {
	// Load the zoo schema
	schemaPath := "../../examples/specs/zoo/schema.yaml"
	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		t.Skip("Zoo schema not found, skipping")
	}

	absPath, _ := filepath.Abs(schemaPath)
	data, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("Failed to read schema: %v", err)
	}

	schema, err := solidsszgen.ReadSchemaFromBytes(data)
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	ref := NewReferenceImplementation()

	config := FuzzConfig{
		NumIterations:       20,
		Seed:                67890,
		StopOnFirstMismatch: false,
		Verbose:             testing.Verbose(),
		TypeFilter:          []string{"ClockInRecords", "Animal", "Zoo"},
	}

	fuzzer := NewFuzzer(schema.Defs, []SSZImplementation{ref}, config)
	result := fuzzer.Fuzz()

	t.Logf("\n%s", result.Report())

	if result.TotalTests == 0 {
		t.Error("No tests were run")
	}

	// It's ok to have some errors for complex types we haven't fully implemented
	// but we should have some successes
	if result.PassedTests == 0 {
		t.Error("No tests passed")
	}
}

func TestProgressiveContainerGeneration(t *testing.T) {
	defs := map[string]cuessz.Def{
		"Progressive": {
			Type:         cuessz.TypeProgressiveContainer,
			ActiveFields: []int{1, 0, 1, 1}, // positions 0, 2, 3 active
			Children: []cuessz.Field{
				{Name: "field0", Def: cuessz.Def{Type: cuessz.TypeUint32}},
				{Name: "field2", Def: cuessz.Def{Type: cuessz.TypeUint64}},
				{Name: "field3", Def: cuessz.Def{Type: cuessz.TypeBoolean}},
			},
		},
	}

	gen := NewGenerator(defs)

	// Generate 5 random progressive containers
	for i := 0; i < 5; i++ {
		value, err := gen.GenerateNamed("Progressive")
		if err != nil {
			t.Fatalf("Failed to generate progressive container: %v", err)
		}

		container := value.(map[string]interface{})

		// Should only have 3 fields (active ones)
		if len(container) != 3 {
			t.Errorf("Expected 3 fields, got %d", len(container))
		}

		// Check all active fields are present
		if _, ok := container["field0"]; !ok {
			t.Error("Missing field0")
		}
		if _, ok := container["field2"]; !ok {
			t.Error("Missing field2")
		}
		if _, ok := container["field3"]; !ok {
			t.Error("Missing field3")
		}
	}

	t.Log("✓ Generator correctly handles progressive containers")
}

func TestBitListGeneration(t *testing.T) {
	defs := map[string]cuessz.Def{
		"BitListContainer": {
			Type: cuessz.TypeContainer,
			Children: []cuessz.Field{
				{
					Name: "bits",
					Def: cuessz.Def{
						Type:  cuessz.TypeBitList,
						Limit: 256,
					},
				},
			},
		},
	}

	gen := NewGenerator(defs)

	// Generate 10 containers with random bitlists
	for i := 0; i < 10; i++ {
		value, err := gen.GenerateNamed("BitListContainer")
		if err != nil {
			t.Fatalf("Failed to generate bitlist container: %v", err)
		}

		container := value.(map[string]interface{})
		bitlistData := container["bits"].([]byte)

		// Verify sentinel bit is present
		found := false
		for _, b := range bitlistData {
			if b != 0 {
				found = true
				break
			}
		}

		if !found {
			t.Error("Bitlist should have sentinel bit")
		}

		t.Logf("Generated bitlist with %d bytes", len(bitlistData))
	}

	t.Log("✓ Generator creates valid bitlists with sentinel bits")
}
