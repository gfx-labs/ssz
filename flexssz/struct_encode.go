package flexssz

import (
	"bytes"
	"fmt"
	"reflect"

	"github.com/holiman/uint256"
)

var (
	// Precalculated types to avoid reflection overhead
	uint256Type = reflect.TypeOf(uint256.Int{})
)

// Marshal encodes a value to SSZ bytes based on its type and struct tags
func Marshal(v any) ([]byte, error) {
	buf := new(bytes.Buffer)
	builder := NewBuilder(buf)

	err := encodeValueToBuilder(builder, v)
	if err != nil {
		return nil, err
	}

	err = builder.Finish()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}


// encodeStructToBuilder encodes a struct using the provided builder
func encodeStructToBuilder(b *Builder, v any) error {
	rv := reflect.ValueOf(v)

	// Handle pointers
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return fmt.Errorf("cannot encode nil pointer")
		}
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("expected struct, got %v", rv.Kind())
	}

	rt := rv.Type()

	// Get type info
	typeInfo, err := GetTypeInfo(rt, nil)
	if err != nil {
		return fmt.Errorf("error getting type info: %w", err)
	}

	// Encode fields in declaration order
	for _, field := range typeInfo.Fields {
		fieldValue := rv.Field(field.Index)

		if field.Type.IsVariable {
			// For variable-size fields, this will write the offset
			err := encodeVariableField(b, fieldValue, field.Type.Tag)
			if err != nil {
				return fmt.Errorf("error encoding variable field %s: %w", field.Name, err)
			}
		} else {
			// For fixed fields, encode directly
			err := encodeFixedField(b, fieldValue, field.Type.Tag)
			if err != nil {
				return fmt.Errorf("error encoding field %s: %w", field.Name, err)
			}
		}
	}

	return nil
}



// encodeFixedField encodes a fixed-size field
func encodeFixedField(b *Builder, v reflect.Value, tag *sszTag) error {
	switch v.Kind() {
	case reflect.Uint8:
		b.EncodeUint8(uint8(v.Uint()))
	case reflect.Uint16:
		b.EncodeUint16(uint16(v.Uint()))
	case reflect.Uint32:
		b.EncodeUint32(uint32(v.Uint()))
	case reflect.Uint64:
		b.EncodeUint64(v.Uint())
	case reflect.Bool:
		b.EncodeBool(v.Bool())
	case reflect.Slice:
		// Fixed-size slices (with ssz-size tag) can be encoded as fixed fields
		if len(tag.Size) > 0 {
			expectedLen := tag.Size[0]
			
			if v.Type().Elem().Kind() == reflect.Uint8 {
				// For bitvector, size is in bits; for regular byte slices, size is in bytes
				if tag.FieldType == "bitvector" {
					expectedBytes := (expectedLen + 7) / 8
					if v.Len() != expectedBytes {
						return fmt.Errorf("bitvector requires %d bytes for %d bits, got %d bytes", expectedBytes, expectedLen, v.Len())
					}
					// Encode as bitvector (no delimiter bit)
					encoded, err := EncodeBitVector(v.Bytes(), expectedLen)  // expectedLen is in bits
					if err != nil {
						return fmt.Errorf("error encoding bitvector: %w", err)
					}
					b.EncodeFixed(encoded)
				} else {
					// Regular byte slice
					if v.Len() != expectedLen {
						return fmt.Errorf("slice length %d does not match ssz-size %d", v.Len(), expectedLen)
					}
					b.EncodeFixed(v.Bytes())
				}
			} else {
				// Other slices - encode each element
				for i := 0; i < v.Len(); i++ {
					elemTag := &sszTag{}
					// For multi-dimensional arrays, pass down the remaining sizes
					if len(tag.Size) > 1 {
						elemTag.Size = tag.Size[1:]
					}
					err := encodeFixedField(b, v.Index(i), elemTag)
					if err != nil {
						return err
					}
				}
			}
		} else {
			// Variable slices cannot be encoded as fixed fields
			return fmt.Errorf("variable slices must be encoded as variable fields")
		}
	case reflect.Array:
		// Check if it's a uint256.Int type (which is [4]uint64)
		if v.Type() == uint256Type {
			// Get the pointer to the uint256.Int
			if v.CanAddr() {
				ptr := v.Addr().Interface().(*uint256.Int)
				if tag.FieldType == "uint128" {
					b.EncodeUint128(ptr)
				} else {
					// Default to uint256
					b.EncodeUint256(ptr)
				}
			} else {
				// If we can't get address, create a copy
				val := v.Interface().(uint256.Int)
				if tag.FieldType == "uint128" {
					b.EncodeUint128(&val)
				} else {
					b.EncodeUint256(&val)
				}
			}
		} else if v.Type().Elem().Kind() == reflect.Uint8 {
			// Byte array
			bytes := make([]byte, v.Len())
			for i := 0; i < v.Len(); i++ {
				bytes[i] = uint8(v.Index(i).Uint())
			}
			b.EncodeFixed(bytes)
		} else {
			// Other arrays - encode each element
			for i := 0; i < v.Len(); i++ {
				err := encodeFixedField(b, v.Index(i), tag)
				if err != nil {
					return err
				}
			}
		}
	case reflect.Ptr:
		// Handle pointer types
		if v.IsNil() {
			return fmt.Errorf("cannot encode nil pointer")
		}
		// Check if it's a pointer to uint256.Int
		if v.Type().Elem() == uint256Type {
			ptr := v.Interface().(*uint256.Int)
			if tag.FieldType == "uint128" {
				b.EncodeUint128(ptr)
			} else {
				// Default to uint256
				b.EncodeUint256(ptr)
			}
		} else {
			// For other pointers, dereference and encode the value
			return encodeFixedField(b, v.Elem(), tag)
		}
	case reflect.Struct:
		// Nested struct
		return encodeStructToBuilder(b, v.Interface())
	default:
		return fmt.Errorf("unsupported type for fixed field: %v", v.Kind())
	}

	return nil
}

// encodeVariableField encodes a variable-size field
func encodeVariableField(b *Builder, v reflect.Value, tag *sszTag) error {
	switch v.Kind() {
	case reflect.String:
		b.EncodeString(v.String())
	case reflect.Slice:
		// Check limit if specified
		if tag.MaxList > 0 && v.Len() > tag.MaxList {
			return fmt.Errorf("slice length %d exceeds limit %d", v.Len(), tag.MaxList)
		}
		
		if v.Type().Elem().Kind() == reflect.Uint8 {
			// Byte slice - check if it's a bitlist
			if tag.FieldType == "bitlist" {
				// Encode as bitlist with delimiter bit
				encoded, err := EncodeBitList(v.Bytes(), tag.MaxList)
				if err != nil {
					return fmt.Errorf("error encoding bitlist: %w", err)
				}
				b.EncodeBytes(encoded)
			} else {
				// Regular byte slice
				b.EncodeBytes(v.Bytes())
			}
		} else {
			// Other slices - enter variable context
			dyn := b.EnterDynamic()
			
			// Get element type info to determine if elements are fixed-size
			elemType := v.Type().Elem()
			elemTag := &sszTag{}
			
			// For lists with ssz-size:"?,32", get the element size from the tag
			if tag != nil && len(tag.Size) > 1 {
				elemTag.Size = tag.Size[1:]
			}
			
			elemTypeInfo, err := GetTypeInfo(elemType, elemTag)
			if err != nil {
				return fmt.Errorf("error getting element type info: %w", err)
			}
			
			// Encode elements based on whether they're fixed or variable
			for i := 0; i < v.Len(); i++ {
				var err error
				if elemTypeInfo.IsVariable {
					err = encodeValue(dyn, v.Index(i), elemTag)
				} else {
					err = encodeFixedField(dyn, v.Index(i), elemTag)
				}
				if err != nil {
					return err
				}
			}
			b = dyn.ExitDynamic()
		}
	case reflect.Struct:
		// Variable-size struct - enter variable context
		dyn := b.EnterDynamic()
		err := encodeStructToBuilder(dyn, v.Interface())
		if err != nil {
			return err
		}
		b = dyn.ExitDynamic()
	case reflect.Ptr:
		// Handle pointer types
		if v.IsNil() {
			return fmt.Errorf("cannot encode nil pointer")
		}
		// For pointers to variable types, encode the pointed value
		return encodeVariableField(b, v.Elem(), tag)
	default:
		return fmt.Errorf("unsupported type for variable field: %v", v.Kind())
	}

	return nil
}

// encodeValue encodes a value based on its type
func encodeValue(b *Builder, v reflect.Value, tag *sszTag) error {
	// Check if value is variable-size
	if typeIsVariable(v.Type(), tag) {
		return encodeVariableField(b, v, tag)
	}
	return encodeFixedField(b, v, tag)
}

// encodeValueToBuilder encodes any value using the provided builder
func encodeValueToBuilder(b *Builder, v any) error {
	rv := reflect.ValueOf(v)

	// Handle pointers by dereferencing
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return fmt.Errorf("cannot encode nil pointer")
		}
		rv = rv.Elem()
	}

	// For structs, use the existing struct encoding logic
	if rv.Kind() == reflect.Struct {
		return encodeStructToBuilder(b, rv.Interface())
	}

	// For other types, use the general encoding logic with an empty tag
	tag := &sszTag{}
	return encodeValue(b, rv, tag)
}

