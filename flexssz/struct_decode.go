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
