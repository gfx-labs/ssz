package flexssz

import (
	"fmt"
	"reflect"
)

// DecodeStruct decodes SSZ bytes into a struct based on struct tags
func DecodeStruct(data []byte, v any) error {
	rv := reflect.ValueOf(v)

	// Must be a pointer to a struct
	if rv.Kind() != reflect.Ptr {
		return fmt.Errorf("v must be a pointer, got %v", rv.Kind())
	}

	if rv.IsNil() {
		return fmt.Errorf("v must not be nil")
	}

	elem := rv.Elem()
	if elem.Kind() != reflect.Struct {
		return fmt.Errorf("v must be a pointer to struct, got pointer to %v", elem.Kind())
	}

	decoder := NewDecoder(data)
	return decodeStructFromDecoder(decoder, elem)
}

// decodeStructFromDecoder decodes a struct using the provided decoder
func decodeStructFromDecoder(dec *Decoder, v reflect.Value) error {
	rt := v.Type()

	// Get type info
	typeInfo, err := GetTypeInfo(rt, nil)
	if err != nil {
		return fmt.Errorf("error getting type info: %w", err)
	}

	// Build container elements
	elements := make([]ContainerElement, 0, len(typeInfo.Fields))

	for _, field := range typeInfo.Fields {
		// Capture field info in closure
		fieldCopy := field
		fieldIndex := field.Index
		fieldName := field.Name

		if field.Type.IsVariable {
			// Variable field
			elements = append(elements, Variable(func(d *Decoder) error {
				fieldValue := v.Field(fieldIndex)
				err := decodeVariableField(d, fieldValue, &fieldCopy)
				if err != nil {
					return fmt.Errorf("error decoding variable field %s: %w", fieldName, err)
				}
				return nil
			}))
		} else {
			// Fixed field
			elements = append(elements, Fixed(func(d *Decoder) error {
				fieldValue := v.Field(fieldIndex)
				err := decodeFixedField(d, fieldValue, &fieldCopy)
				if err != nil {
					return fmt.Errorf("error decoding field %s: %w", fieldName, err)
				}
				return nil
			}))
		}
	}

	// Decode container
	return dec.DecodeContainer(elements...)
}



// decodeValue decodes a value based on its type
func decodeValue(d *Decoder, v reflect.Value, fieldInfo *FieldInfo) error {
	// Check if value is variable-size
	if fieldInfo.Type.IsVariable {
		return decodeVariableField(d, v, fieldInfo)
	}
	return decodeFixedField(d, v, fieldInfo)
}

// getFixedSize returns the size in bytes of a fixed-size type
func getFixedSize(t reflect.Type, tag *sszTag) (int, error) {
	switch t.Kind() {
	case reflect.Uint8:
		return 1, nil
	case reflect.Uint16:
		return 2, nil
	case reflect.Uint32:
		return 4, nil
	case reflect.Uint64:
		return 8, nil
	case reflect.Bool:
		return 1, nil
	case reflect.Slice:
		// Fixed-size slices (with ssz-size tag)
		if tag != nil && len(tag.Size) > 0 {
			// For bitvector, size is in bits
			if tag.FieldType == "bitvector" && t.Elem().Kind() == reflect.Uint8 {
				// Convert bits to bytes
				return (tag.Size[0] + 7) / 8, nil
			}
			// Calculate total size based on element size and count
			elemTag := &sszTag{}
			if len(tag.Size) > 1 {
				elemTag.Size = tag.Size[1:]
			}
			elemSize, err := getFixedSize(t.Elem(), elemTag)
			if err != nil {
				return 0, err
			}
			return elemSize * tag.Size[0], nil
		}
		return 0, fmt.Errorf("cannot get fixed size for variable slice")
	case reflect.Array:
		if t == uint256Type {
			if tag != nil && tag.FieldType == "uint128" {
				return 16, nil
			}
			return 32, nil
		}
		// Array size is element size * length
		elemTag := &sszTag{}
		elemSize, err := getFixedSize(t.Elem(), elemTag)
		if err != nil {
			return 0, err
		}
		return elemSize * t.Len(), nil
	case reflect.Ptr:
		// For pointers, get the size of the pointed type
		if t.Elem() == uint256Type {
			if tag != nil && tag.FieldType == "uint128" {
				return 16, nil
			}
			return 32, nil
		}
		return getFixedSize(t.Elem(), tag)
	case reflect.Struct:
		// For structs, we need to calculate the total fixed size
		typeInfo, err := GetTypeInfo(t, tag)
		if err != nil {
			return 0, err
		}

		if typeInfo.IsVariable {
			return -1, nil
		}
		return typeInfo.FixedSize, nil
	default:
		return 0, fmt.Errorf("cannot get fixed size for type %v", t)
	}
}

