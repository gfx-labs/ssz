package ssz

import "fmt"

type TypeName string

const (
	TypeUint8   TypeName = "uint8"
	TypeUint16  TypeName = "uint16"
	TypeUint32  TypeName = "uint32"
	TypeUint64  TypeName = "uint64"
	TypeUint128 TypeName = "uint128"
	TypeUint256 TypeName = "uint256"

	TypeBoolean TypeName = "boolean"

	TypeContainer TypeName = "container"

	TypeVector TypeName = "vector"
	TypeList   TypeName = "list"

	TypeBitVector TypeName = "bitvector"
	TypeBitList   TypeName = "bitlist"

	TypeUnion TypeName = "union"

	// This is a special type that is not an ssz type, but rather a ref to another type in the schema
	TypeRef TypeName = "ref"
)

type Field struct {
	Name string   `json:"name"`
	Type TypeName `json:"type"`

	Size  uint64 `json:"size,omitempty"`
	Limit uint64 `json:"limit,omitempty"`

	Ref      string  `json:"ref,omitempty"`
	Children []Field `json:"children,omitempty"`
}

// IsVariable determines if a field is variable-size
func (f *Field) IsVariable(refs map[string]Field) (bool, error) {
	const maxIterations = 1000 // Sanity check to prevent infinite recursion
	return isVariable(f, refs, 0, maxIterations)
}

// isVariable is the internal implementation with iteration tracking
func isVariable(f *Field, refs map[string]Field, iterations, maxIterations int) (bool, error) {
	if iterations >= maxIterations {
		return false, fmt.Errorf("max iterations reached while checking IsVariable - possible circular reference")
	}

	switch f.Type {
	case TypeList, TypeBitList, TypeUnion:
		return true, nil
	case TypeContainer, TypeVector, TypeBitVector:
		for _, child := range f.Children {
			isVar, err := isVariable(&child, refs, iterations+1, maxIterations)
			if err != nil {
				return false, err
			}
			if isVar {
				return true, nil
			}
		}
	case TypeRef:
		if f.Ref == "" {
			return false, fmt.Errorf("field has type 'ref' but no ref specified")
		}
		refField, ok := refs[f.Ref]
		if !ok {
			return false, fmt.Errorf("ref type '%s' not found", f.Ref)
		}
		return isVariable(&refField, refs, iterations+1, maxIterations)
	}
	return false, nil
}

// IsValid validates the field and all its subfields
func (f *Field) IsValid(refs map[string]Field) error {
	const maxIterations = 1000 // Sanity check to prevent infinite recursion
	return isValid(f, refs, 0, maxIterations)
}

// isValid is the internal implementation with iteration tracking
func isValid(f *Field, refs map[string]Field, iterations, maxIterations int) error {
	if iterations >= maxIterations {
		return fmt.Errorf("max iterations reached while validating field '%s' - possible circular reference", f.Name)
	}

	// Validate field name
	if f.Name == "" {
		return fmt.Errorf("field name cannot be empty")
	}

	// Validate based on type
	switch f.Type {
	case TypeUint8, TypeUint16, TypeUint32, TypeUint64, TypeUint128, TypeUint256, TypeBoolean:
		// Basic types are always valid
		return nil


	case TypeVector, TypeBitVector:
		// Fixed-size types must have Size specified
		if f.Size == 0 {
			return fmt.Errorf("field '%s' of type '%s' must have non-zero size", f.Name, f.Type)
		}
		// Validate children for container vectors
		if f.Type == TypeVector && len(f.Children) > 0 {
			for i, child := range f.Children {
				if err := isValid(&child, refs, iterations+1, maxIterations); err != nil {
					return fmt.Errorf("field '%s' child[%d]: %w", f.Name, i, err)
				}
			}
		}
		return nil

	case TypeList, TypeBitList:
		// Variable-size types must have Limit specified
		if f.Limit == 0 {
			return fmt.Errorf("field '%s' of type '%s' must have non-zero limit", f.Name, f.Type)
		}
		// Validate children for container lists
		if f.Type == TypeList && len(f.Children) > 0 {
			for i, child := range f.Children {
				if err := isValid(&child, refs, iterations+1, maxIterations); err != nil {
					return fmt.Errorf("field '%s' child[%d]: %w", f.Name, i, err)
				}
			}
		}
		return nil

	case TypeContainer:
		// Containers must have children
		if len(f.Children) == 0 {
			return fmt.Errorf("field '%s' of type 'container' must have children", f.Name)
		}
		// Validate all children
		for i, child := range f.Children {
			if err := isValid(&child, refs, iterations+1, maxIterations); err != nil {
				return fmt.Errorf("field '%s' child[%d]: %w", f.Name, i, err)
			}
		}
		return nil

	case TypeUnion:
		// Unions must have children
		if len(f.Children) == 0 {
			return fmt.Errorf("field '%s' of type 'union' must have children", f.Name)
		}
		// Validate all children
		for i, child := range f.Children {
			if err := isValid(&child, refs, iterations+1, maxIterations); err != nil {
				return fmt.Errorf("field '%s' child[%d]: %w", f.Name, i, err)
			}
		}
		return nil

	case TypeRef:
		// Refs must have a reference
		if f.Ref == "" {
			return fmt.Errorf("field '%s' has type 'ref' but no ref specified", f.Name)
		}
		// Check if ref exists
		refField, ok := refs[f.Ref]
		if !ok {
			return fmt.Errorf("field '%s' references type '%s' which is not found", f.Name, f.Ref)
		}
		// Validate the referenced field
		return isValid(&refField, refs, iterations+1, maxIterations)

	default:
		return fmt.Errorf("field '%s' has unknown type '%s'", f.Name, f.Type)
	}
}
