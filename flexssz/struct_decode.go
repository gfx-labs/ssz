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
func decodeStructFromDecoder(d *Decoder, v reflect.Value) error {
	rt := v.Type()
	
	// Get type info
	typeInfo, err := GetTypeInfo(rt, nil)
	if err != nil {
		return fmt.Errorf("error getting type info: %w", err)
	}
	
	// Calculate fixed part size and find variable fields
	fixedSize := 0
	var variableFields []int
	for i, field := range typeInfo.Fields {
		if field.Type.IsVariable {
			fixedSize += 4 // Offset size
			variableFields = append(variableFields, i)
		} else {
			fixedSize += field.Type.FixedSize
		}
	}
	
	// Read fixed part
	fixedData, err := d.ReadN(fixedSize)
	if err != nil {
		return fmt.Errorf("error reading fixed part: %w", err)
	}
	fixedDecoder := NewDecoder(fixedData)
	
	// Decode fields
	variableIndex := 0
	for i, field := range typeInfo.Fields {
		fieldValue := v.Field(field.Index)
		
		if field.Type.IsVariable {
			// Read offset
			offset, err := fixedDecoder.ReadUint32()
			if err != nil {
				return fmt.Errorf("error reading offset for field %s: %w", field.Name, err)
			}
			
			// Calculate size
			var size int
			if variableIndex < len(variableFields)-1 {
				// Not the last variable field - peek at next variable field's offset
				// We need to skip ahead to find it
				nextFieldIndex := variableFields[variableIndex+1]
				savedPos := fixedDecoder.cur
				
				// Skip to the next variable field's offset position
				skipBytes := 0
				for j := i + 1; j < nextFieldIndex; j++ {
					if !typeInfo.Fields[j].Type.IsVariable {
						skipBytes += typeInfo.Fields[j].Type.FixedSize
					}
				}
				
				if skipBytes > 0 {
					_, err := fixedDecoder.ReadN(skipBytes)
					if err != nil {
						return fmt.Errorf("error skipping to next offset: %w", err)
					}
				}
				
				nextOffset, err := fixedDecoder.ReadUint32()
				if err != nil {
					return fmt.Errorf("error reading next offset: %w", err)
				}
				
				// Restore position
				fixedDecoder.cur = savedPos
				
				size = int(nextOffset - offset)
			} else {
				// Last variable field - remaining data
				size = len(d.xs) - int(offset)
			}
			
			// Read variable data
			var varData []byte
			if size > 0 {
				varData, err = d.ReadN(size)
				if err != nil {
					return fmt.Errorf("error reading variable data for field %s: %w", field.Name, err)
				}
			} else {
				// Empty variable field
				varData = []byte{}
			}
			
			// Decode variable field
			varDecoder := NewDecoder(varData)
			err = decodeVariableField(varDecoder, fieldValue, field.Type.Tag)
			if err != nil {
				return fmt.Errorf("error decoding variable field %s: %w", field.Name, err)
			}
			
			variableIndex++
		} else {
			// Decode fixed field directly from fixed decoder
			err := decodeFixedField(fixedDecoder, fieldValue, field.Type.Tag)
			if err != nil {
				return fmt.Errorf("error decoding field %s: %w", field.Name, err)
			}
		}
	}
	
	return nil
}

// peekUint32 reads a uint32 without advancing the decoder
func peekUint32(d *Decoder) (uint32, error) {
	// Save current position
	data := d.Remaining()
	if len(data) < 4 {
		return 0, fmt.Errorf("insufficient data for uint32")
	}
	
	// Read value
	val := uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16 | uint32(data[3])<<24
	
	return val, nil
}

// decodeFixedField decodes a fixed-size field
func decodeFixedField(d *Decoder, v reflect.Value, tag *sszTag) error {
	switch v.Kind() {
	case reflect.Uint8:
		val, err := d.ReadUint8()
		if err != nil {
			return err
		}
		v.SetUint(uint64(val))
	case reflect.Uint16:
		val, err := d.ReadUint16()
		if err != nil {
			return err
		}
		v.SetUint(uint64(val))
	case reflect.Uint32:
		val, err := d.ReadUint32()
		if err != nil {
			return err
		}
		v.SetUint(uint64(val))
	case reflect.Uint64:
		val, err := d.ReadUint64()
		if err != nil {
			return err
		}
		v.SetUint(val)
	case reflect.Bool:
		val, err := d.ReadBool()
		if err != nil {
			return err
		}
		v.SetBool(val)
	case reflect.Slice:
		// Fixed-size slices (with ssz-size tag)
		if len(tag.Size) > 0 {
			expectedLen := tag.Size[0]
			
			if v.Type().Elem().Kind() == reflect.Uint8 {
				// For bitvector, size is in bits; for regular byte slices, size is in bytes
				var bytesToRead int
				if tag.FieldType == "bitvector" {
					bytesToRead = (expectedLen + 7) / 8  // Convert bits to bytes
				} else {
					bytesToRead = expectedLen
				}
				
				// Byte slice
				bytes, err := d.ReadN(bytesToRead)
				if err != nil {
					return err
				}
				
				// Handle bitvector
				if tag.FieldType == "bitvector" {
					// Decode bitvector (validate no extra bits)
					decoded, err := DecodeBitVector(bytes, expectedLen)  // expectedLen is in bits
					if err != nil {
						return fmt.Errorf("error decoding bitvector: %w", err)
					}
					v.SetBytes(decoded)
				} else {
					v.SetBytes(bytes)
				}
			} else {
				// Other slices
				slice := reflect.MakeSlice(v.Type(), expectedLen, expectedLen)
				
				// Decode each element
				for i := 0; i < expectedLen; i++ {
					elemTag := &sszTag{}
					// For multi-dimensional arrays, pass down the remaining sizes
					if len(tag.Size) > 1 {
						elemTag.Size = tag.Size[1:]
					}
					err := decodeFixedField(d, slice.Index(i), elemTag)
					if err != nil {
						return err
					}
				}
				
				v.Set(slice)
			}
		} else {
			return fmt.Errorf("slice in fixed field must have ssz-size tag")
		}
	case reflect.Array:
		// Check if it's a uint256.Int type
		if v.Type() == uint256Type {
			if tag.FieldType == "uint128" {
				val, err := d.ReadUint128()
				if err != nil {
					return err
				}
				v.Set(reflect.ValueOf(*val))
			} else {
				// Default to uint256
				val, err := d.ReadUint256()
				if err != nil {
					return err
				}
				v.Set(reflect.ValueOf(*val))
			}
		} else if v.Type().Elem().Kind() == reflect.Uint8 {
			// Byte array
			bytes, err := d.ReadN(v.Len())
			if err != nil {
				return err
			}
			for i := 0; i < v.Len(); i++ {
				v.Index(i).SetUint(uint64(bytes[i]))
			}
		} else {
			// Other arrays - decode each element
			for i := 0; i < v.Len(); i++ {
				err := decodeFixedField(d, v.Index(i), tag)
				if err != nil {
					return err
				}
			}
		}
	case reflect.Ptr:
		// Handle pointer types
		if v.Type().Elem() == uint256Type {
			// Allocate new uint256.Int
			if v.IsNil() {
				v.Set(reflect.New(uint256Type))
			}
			// Decode into the pointed value
			if tag.FieldType == "uint128" {
				val, err := d.ReadUint128()
				if err != nil {
					return err
				}
				v.Elem().Set(reflect.ValueOf(*val))
			} else {
				// Default to uint256
				val, err := d.ReadUint256()
				if err != nil {
					return err
				}
				v.Elem().Set(reflect.ValueOf(*val))
			}
		} else {
			// For other pointers, ensure it's allocated then decode into it
			if v.IsNil() {
				v.Set(reflect.New(v.Type().Elem()))
			}
			return decodeFixedField(d, v.Elem(), tag)
		}
	case reflect.Struct:
		// Nested struct
		return decodeStructFromDecoder(d, v)
	default:
		return fmt.Errorf("unsupported type for fixed field: %v", v.Kind())
	}
	
	return nil
}

// decodeVariableField decodes a variable-size field
func decodeVariableField(d *Decoder, v reflect.Value, tag *sszTag) error {
	switch v.Kind() {
	case reflect.String:
		// Read all remaining bytes and convert to string
		buf, err := d.ReadAll()
		if err != nil {
			return err
		}
		v.SetString(string(buf))
	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			// Byte slice - read all remaining bytes
			bytes, err := d.ReadAll()
			if err != nil {
				return err
			}
			
			// Handle bitlist
			if tag.FieldType == "bitlist" {
				// Decode bitlist (remove delimiter bit)
				decoded, numBits, err := DecodeBitList(bytes, tag.MaxList)
				if err != nil {
					return fmt.Errorf("error decoding bitlist: %w", err)
				}
				// For now, we store the full bytes including padding
				// The actual number of bits can be tracked separately if needed
				_ = numBits
				v.SetBytes(decoded)
			} else {
				// Check limit for regular byte slices
				if tag.MaxList > 0 && len(bytes) > tag.MaxList {
					return fmt.Errorf("slice length %d exceeds limit %d", len(bytes), tag.MaxList)
				}
				v.SetBytes(bytes)
			}
		} else {
			// Other slices - need to know element size
			elemType := v.Type().Elem()
			elemTag := &sszTag{} // Elements don't have their own tags
			
			// For variable-size elements, we need to handle offsets
			if typeIsVariable(elemType, elemTag) {
				// Read offsets first
				remaining := d.Remaining()
				if len(remaining) == 0 {
					// Empty slice
					v.Set(reflect.MakeSlice(v.Type(), 0, 0))
					return nil
				}
				
				if len(remaining) < 4 {
					return fmt.Errorf("invalid data for variable-size slice: less than 4 bytes")
				}
				
				// Read first offset to determine number of elements
				firstOffset, err := d.ReadUint32()
				if err != nil {
					return err
				}
				
				// Handle empty slice (no offsets means no elements)
				if firstOffset == 0 {
					v.Set(reflect.MakeSlice(v.Type(), 0, 0))
					return nil
				}
				
				numElements := int(firstOffset) / 4
				offsets := make([]uint32, numElements)
				offsets[0] = firstOffset
				
				for i := 1; i < numElements; i++ {
					offset, err := d.ReadUint32()
					if err != nil {
						return err
					}
					offsets[i] = offset
				}
				
				// Check limit
				if tag.MaxList > 0 && numElements > tag.MaxList {
					return fmt.Errorf("slice length %d exceeds limit %d", numElements, tag.MaxList)
				}
				
				// Create slice
				slice := reflect.MakeSlice(v.Type(), numElements, numElements)
				
				// Decode each element
				for i := 0; i < numElements; i++ {
					var size int
					if i < numElements-1 {
						size = int(offsets[i+1] - offsets[i])
					} else {
						// For the last element, we need to calculate from the total remaining data
						// The total data starts after all offsets (4 * numElements bytes)
						totalDataSize := len(d.xs) - 4*numElements
						previousDataSize := int(offsets[i]) - 4*numElements
						size = totalDataSize - previousDataSize
					}
					
					elemData, err := d.ReadN(size)
					if err != nil {
						return err
					}
					
					elemDecoder := NewDecoder(elemData)
					err = decodeValue(elemDecoder, slice.Index(i), elemTag)
					if err != nil {
						return err
					}
				}
				
				v.Set(slice)
			} else {
				// Fixed-size elements
				elemSize, err := getFixedSize(elemType, elemTag)
				if err != nil {
					return err
				}
				
				remaining := len(d.Remaining())
				numElements := remaining / elemSize
				
				if remaining%elemSize != 0 {
					return fmt.Errorf("invalid data size for slice: %d bytes cannot be divided by element size %d", remaining, elemSize)
				}
				
				// Check limit
				if tag.MaxList > 0 && numElements > tag.MaxList {
					return fmt.Errorf("slice length %d exceeds limit %d", numElements, tag.MaxList)
				}
				
				// Create slice
				slice := reflect.MakeSlice(v.Type(), numElements, numElements)
				
				// Decode each element
				for i := 0; i < numElements; i++ {
					err := decodeFixedField(d, slice.Index(i), elemTag)
					if err != nil {
						return err
					}
				}
				
				v.Set(slice)
			}
		}
	case reflect.Struct:
		// Variable-size struct
		return decodeStructFromDecoder(d, v)
	case reflect.Ptr:
		// Handle pointer types
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		// Decode into the pointed value
		return decodeVariableField(d, v.Elem(), tag)
	default:
		return fmt.Errorf("unsupported type for variable field: %v", v.Kind())
	}
	
	return nil
}

// decodeValue decodes a value based on its type
func decodeValue(d *Decoder, v reflect.Value, tag *sszTag) error {
	// Check if value is variable-size
	if typeIsVariable(v.Type(), tag) {
		return decodeVariableField(d, v, tag)
	}
	return decodeFixedField(d, v, tag)
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