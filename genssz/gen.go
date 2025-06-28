package genssz

import (
	"fmt"

	"github.com/gfx-labs/ssz"
	"sigs.k8s.io/yaml"
)

type Field struct {
	Name     string        `yaml:"name"`
	Type     ssz.TypeName  `yaml:"type"`
	Size     uint64        `yaml:"size,omitempty"`
	Limit    uint64        `yaml:"limit,omitempty"`
	Ref      string        `yaml:"ref,omitempty"`
	Children []Field       `yaml:"children,omitempty"`
}

// ToSSZField converts Field to ssz.Field, handling bytevector alias
func (f Field) ToSSZField() ssz.Field {
	// Handle bytevector alias - convert to vector of uint8
	if f.Type == "bytevector" {
		return ssz.Field{
			Name: f.Name,
			Type: ssz.TypeVector,
			Size: f.Size,
			Children: []ssz.Field{
				{
					Name: "element",
					Type: ssz.TypeUint8,
				},
			},
		}
	}
	
	// For other types, convert normally
	result := ssz.Field{
		Name:  f.Name,
		Type:  f.Type,
		Size:  f.Size,
		Limit: f.Limit,
		Ref:   f.Ref,
	}
	
	// Convert children recursively
	if len(f.Children) > 0 {
		result.Children = make([]ssz.Field, len(f.Children))
		for i, child := range f.Children {
			result.Children[i] = child.ToSSZField()
		}
	}
	
	return result
}

type Schema struct {
	Package string  `yaml:"package"`
	Structs []Field `yaml:"structs"`
}

type World struct {
	Types map[string]Type
}

type Type struct {
	Name string
	Type string

	Ref string

	Variable *VariableType
	Fixed    *FixedType
}

type VariableType struct {
	Limit uint64
}

type FixedType struct {
	Size uint64
}

// ReadSchemaFromBytes reads a schema from YAML bytes and returns a Schema
func ReadSchemaFromBytes(data []byte) (*Schema, error) {
	var schema Schema
	err := yaml.Unmarshal(data, &schema)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml: %w", err)
	}
	return &schema, nil
}

// ParseSchemaToWorld converts a Schema into a World representation
func ParseSchemaToWorld(schema *Schema) (*World, error) {
	world := &World{
		Types: make(map[string]Type),
	}

	// Process only top-level structs in the schema
	for _, field := range schema.Structs {
		typ := Type{
			Name: field.Name,
			Type: string(field.Type),
		}

		switch field.Type {
		case ssz.TypeRef:
			typ.Ref = field.Ref
		case ssz.TypeList, ssz.TypeBitList:
			typ.Variable = &VariableType{
				Limit: field.Limit,
			}
		case ssz.TypeVector, ssz.TypeBitVector:
			typ.Fixed = &FixedType{
				Size: field.Size,
			}
		}

		// Add only the top-level type to the world
		world.Types[field.Name] = typ
	}

	return world, nil
}
