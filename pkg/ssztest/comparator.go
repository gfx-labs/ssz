package ssztest

import (
	"bytes"
	"fmt"

	"github.com/gfx-labs/cuessz"
)

// SSZImplementation represents a specific SSZ library implementation
type SSZImplementation interface {
	// Name returns the implementation name
	Name() string

	// Marshal serializes a value to SSZ bytes
	Marshal(value interface{}, def cuessz.Def) ([]byte, error)

	// Unmarshal deserializes SSZ bytes to a value
	Unmarshal(data []byte, def cuessz.Def) (interface{}, error)

	// HashTreeRoot computes the merkle root (optional, can return nil if not supported)
	HashTreeRoot(value interface{}, def cuessz.Def) ([]byte, error)
}

// Comparator compares results from multiple SSZ implementations
type Comparator struct {
	impls []SSZImplementation
	defs  map[string]cuessz.Def
}

// NewComparator creates a new cross-implementation comparator
func NewComparator(defs map[string]cuessz.Def, impls ...SSZImplementation) *Comparator {
	return &Comparator{
		impls: impls,
		defs:  defs,
	}
}

// ComparisonResult holds the results from comparing implementations
type ComparisonResult struct {
	TypeName        string
	InputValue      interface{}
	SerializedData  map[string][]byte // impl name -> serialized bytes
	DeserializedVal map[string]interface{}
	HashTreeRoots   map[string][]byte
	Errors          map[string]error
	Mismatches      []Mismatch
}

// Mismatch describes a difference between implementations
type Mismatch struct {
	Type        string // "serialization", "deserialization", "hash_tree_root"
	Impl1       string
	Impl2       string
	Description string
	Data1       []byte
	Data2       []byte
}

// Compare runs a value through all implementations and compares results
func (c *Comparator) Compare(typeName string, value interface{}) (*ComparisonResult, error) {
	def, ok := c.defs[typeName]
	if !ok {
		return nil, fmt.Errorf("unknown type: %s", typeName)
	}

	result := &ComparisonResult{
		TypeName:        typeName,
		InputValue:      value,
		SerializedData:  make(map[string][]byte),
		DeserializedVal: make(map[string]interface{}),
		HashTreeRoots:   make(map[string][]byte),
		Errors:          make(map[string]error),
		Mismatches:      make([]Mismatch, 0),
	}

	// Step 1: Marshal with all implementations
	for _, impl := range c.impls {
		data, err := impl.Marshal(value, def)
		if err != nil {
			result.Errors[impl.Name()+".Marshal"] = err
			continue
		}
		result.SerializedData[impl.Name()] = data
	}

	// Step 2: Compare serialization results
	c.compareSerializations(result)

	// Step 3: Unmarshal with all implementations (using their own serialized data)
	for _, impl := range c.impls {
		data, ok := result.SerializedData[impl.Name()]
		if !ok {
			continue
		}

		val, err := impl.Unmarshal(data, def)
		if err != nil {
			result.Errors[impl.Name()+".Unmarshal"] = err
			continue
		}
		result.DeserializedVal[impl.Name()] = val
	}

	// Step 4: Compute hash tree roots
	for _, impl := range c.impls {
		root, err := impl.HashTreeRoot(value, def)
		if err != nil {
			// Hash tree root is optional, don't record as error if not supported
			if err.Error() != "not implemented" {
				result.Errors[impl.Name()+".HashTreeRoot"] = err
			}
			continue
		}
		if root != nil {
			result.HashTreeRoots[impl.Name()] = root
		}
	}

	// Step 5: Compare hash tree roots
	c.compareHashTreeRoots(result)

	return result, nil
}

func (c *Comparator) compareSerializations(result *ComparisonResult) {
	impls := make([]string, 0, len(result.SerializedData))
	for impl := range result.SerializedData {
		impls = append(impls, impl)
	}

	for i := 0; i < len(impls); i++ {
		for j := i + 1; j < len(impls); j++ {
			impl1, impl2 := impls[i], impls[j]
			data1, data2 := result.SerializedData[impl1], result.SerializedData[impl2]

			if !bytes.Equal(data1, data2) {
				result.Mismatches = append(result.Mismatches, Mismatch{
					Type:        "serialization",
					Impl1:       impl1,
					Impl2:       impl2,
					Description: fmt.Sprintf("Serialized data differs: %d bytes vs %d bytes", len(data1), len(data2)),
					Data1:       data1,
					Data2:       data2,
				})
			}
		}
	}
}

func (c *Comparator) compareHashTreeRoots(result *ComparisonResult) {
	impls := make([]string, 0, len(result.HashTreeRoots))
	for impl := range result.HashTreeRoots {
		impls = append(impls, impl)
	}

	for i := 0; i < len(impls); i++ {
		for j := i + 1; j < len(impls); j++ {
			impl1, impl2 := impls[i], impls[j]
			root1, root2 := result.HashTreeRoots[impl1], result.HashTreeRoots[impl2]

			if !bytes.Equal(root1, root2) {
				result.Mismatches = append(result.Mismatches, Mismatch{
					Type:        "hash_tree_root",
					Impl1:       impl1,
					Impl2:       impl2,
					Description: "Hash tree roots differ",
					Data1:       root1,
					Data2:       root2,
				})
			}
		}
	}
}

// HasMismatches returns true if any mismatches were found
func (r *ComparisonResult) HasMismatches() bool {
	return len(r.Mismatches) > 0
}

// HasErrors returns true if any errors occurred
func (r *ComparisonResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// Report generates a human-readable report
func (r *ComparisonResult) Report() string {
	var report string

	report += fmt.Sprintf("Type: %s\n", r.TypeName)
	report += fmt.Sprintf("Implementations tested: %d\n", len(r.SerializedData))

	if len(r.Errors) > 0 {
		report += "\nErrors:\n"
		for name, err := range r.Errors {
			report += fmt.Sprintf("  %s: %v\n", name, err)
		}
	}

	if len(r.Mismatches) > 0 {
		report += "\nMismatches:\n"
		for i, m := range r.Mismatches {
			report += fmt.Sprintf("  [%d] %s mismatch between %s and %s\n", i+1, m.Type, m.Impl1, m.Impl2)
			report += fmt.Sprintf("      %s\n", m.Description)
			if m.Type == "serialization" {
				report += fmt.Sprintf("      %s: %x...\n", m.Impl1, truncate(m.Data1, 32))
				report += fmt.Sprintf("      %s: %x...\n", m.Impl2, truncate(m.Data2, 32))
			} else if m.Type == "hash_tree_root" {
				report += fmt.Sprintf("      %s: %x\n", m.Impl1, m.Data1)
				report += fmt.Sprintf("      %s: %x\n", m.Impl2, m.Data2)
			}
		}
	} else if len(r.Errors) == 0 {
		report += "\n✓ All implementations agree!\n"
	}

	return report
}

func truncate(data []byte, maxLen int) []byte {
	if len(data) <= maxLen {
		return data
	}
	return data[:maxLen]
}
