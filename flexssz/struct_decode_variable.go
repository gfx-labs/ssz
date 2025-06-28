package flexssz

import (
	"fmt"
	"reflect"

	"github.com/gfx-labs/ssz"
)

// decodeVariableField decodes a variable-size field
func decodeVariableField(d *Decoder, v reflect.Value, fieldInfo *FieldInfo) error {
	// Handle pointer types first
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		// Decode into the dereferenced value
		return decodeVariableField(d, v.Elem(), fieldInfo)
	}

	// Switch on SSZ type
	switch fieldInfo.Type.Type {
	case ssz.TypeList:
		return decodeList(d, v, fieldInfo)
	case ssz.TypeBitList:
		return decodeBitList(d, v, fieldInfo)
	case ssz.TypeContainer:
		return decodeVariableContainer(d, v, fieldInfo)
	default:
		return fmt.Errorf("unsupported SSZ type for variable field: %v", fieldInfo.Type.Type)
	}
}

// decodeList decodes a variable-size list
func decodeList(d *Decoder, v reflect.Value, fieldInfo *FieldInfo) error {
	// Special handling for strings (list of bytes in SSZ)
	if v.Kind() == reflect.String {
		return decodeString(d, v, fieldInfo)
	}

	if v.Kind() != reflect.Slice {
		return fmt.Errorf("cannot decode list into %v", v.Kind())
	}

	// Special case for byte slices
	if v.Type().Elem().Kind() == reflect.Uint8 {
		return decodeByteSlice(d, v, fieldInfo)
	}

	// General case for other slice types
	return decodeSlice(d, v, fieldInfo)
}

// decodeString decodes a string (which is a list of bytes in SSZ)
func decodeString(d *Decoder, v reflect.Value, fieldInfo *FieldInfo) error {
	if v.Kind() != reflect.String {
		return fmt.Errorf("cannot decode string into %v", v.Kind())
	}

	// Read all remaining bytes and convert to string
	buf, err := d.ReadAll()
	if err != nil {
		return err
	}
	v.SetString(string(buf))
	return nil
}

// decodeByteSlice decodes a byte slice
func decodeByteSlice(d *Decoder, v reflect.Value, fieldInfo *FieldInfo) error {
	if v.Kind() != reflect.Slice || v.Type().Elem().Kind() != reflect.Uint8 {
		return fmt.Errorf("cannot decode byte slice into %v", v.Type())
	}

	// Read all remaining bytes
	bytes, err := d.ReadAll()
	if err != nil {
		return err
	}

	// Check limit if specified
	tag := fieldInfo.Type.Tag
	if tag != nil && tag.MaxList > 0 && len(bytes) > tag.MaxList {
		return fmt.Errorf("slice length %d exceeds limit %d", len(bytes), tag.MaxList)
	}

	v.SetBytes(bytes)
	return nil
}

// decodeSlice decodes a general slice (not bytes)
func decodeSlice(d *Decoder, v reflect.Value, fieldInfo *FieldInfo) error {
	if v.Kind() != reflect.Slice {
		return fmt.Errorf("cannot decode slice into %v", v.Kind())
	}

	elemType := v.Type().Elem()
	elemTag := &sszTag{} // Elements don't have their own tags
	
	// Get type info for elements
	elemTypeInfo, err := GetTypeInfo(elemType, elemTag)
	if err != nil {
		return fmt.Errorf("error getting element type info: %w", err)
	}

	// For variable-size elements, we need to handle offsets
	if elemTypeInfo.IsVariable {
		return decodeVariableElementSlice(d, v, fieldInfo, elemTypeInfo)
	} else {
		return decodeFixedElementSlice(d, v, fieldInfo, elemTypeInfo)
	}
}

// decodeVariableElementSlice decodes a slice with variable-size elements
func decodeVariableElementSlice(d *Decoder, v reflect.Value, fieldInfo *FieldInfo, elemTypeInfo *TypeInfo) error {
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
	tag := fieldInfo.Type.Tag
	if tag != nil && tag.MaxList > 0 && numElements > tag.MaxList {
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
		// Create a temporary FieldInfo for the element
		elemFieldInfo := &FieldInfo{
			Type: elemTypeInfo,
			Name: fmt.Sprintf("%s[%d]", fieldInfo.Name, i),
		}
		err = decodeValue(elemDecoder, slice.Index(i), elemFieldInfo)
		if err != nil {
			return err
		}
	}

	v.Set(slice)
	return nil
}

// decodeFixedElementSlice decodes a slice with fixed-size elements
func decodeFixedElementSlice(d *Decoder, v reflect.Value, fieldInfo *FieldInfo, elemTypeInfo *TypeInfo) error {
	elemSize := elemTypeInfo.FixedSize
	if elemSize <= 0 {
		return fmt.Errorf("fixed element type has invalid size: %d", elemSize)
	}

	remaining := len(d.Remaining())
	numElements := remaining / elemSize

	if remaining%elemSize != 0 {
		return fmt.Errorf("invalid data size for slice: %d bytes cannot be divided by element size %d", remaining, elemSize)
	}

	// Check limit
	tag := fieldInfo.Type.Tag
	if tag != nil && tag.MaxList > 0 && numElements > tag.MaxList {
		return fmt.Errorf("slice length %d exceeds limit %d", numElements, tag.MaxList)
	}

	// Create slice
	slice := reflect.MakeSlice(v.Type(), numElements, numElements)

	// Decode each element
	for i := 0; i < numElements; i++ {
		// Create a temporary FieldInfo for the element
		elemFieldInfo := &FieldInfo{
			Type: elemTypeInfo,
			Name: fmt.Sprintf("%s[%d]", fieldInfo.Name, i),
		}
		err := decodeFixedField(d, slice.Index(i), elemFieldInfo)
		if err != nil {
			return err
		}
	}

	v.Set(slice)
	return nil
}

// decodeBitList decodes a variable-size bitlist
func decodeBitList(d *Decoder, v reflect.Value, fieldInfo *FieldInfo) error {
	if v.Kind() != reflect.Slice || v.Type().Elem().Kind() != reflect.Uint8 {
		return fmt.Errorf("cannot decode bitlist into %v (expected []byte)", v.Type())
	}

	// Read all remaining bytes
	bytes, err := d.ReadAll()
	if err != nil {
		return err
	}

	tag := fieldInfo.Type.Tag
	maxBits := 0
	if tag != nil {
		maxBits = tag.MaxList
	}

	// Decode bitlist (remove delimiter bit)
	decoded, numBits, err := DecodeBitList(bytes, maxBits)
	if err != nil {
		return fmt.Errorf("error decoding bitlist: %w", err)
	}

	// For now, we store the full bytes including padding
	// The actual number of bits can be tracked separately if needed
	_ = numBits
	v.SetBytes(decoded)
	return nil
}

// decodeVariableContainer decodes a variable-size container (struct)
func decodeVariableContainer(d *Decoder, v reflect.Value, fieldInfo *FieldInfo) error {
	switch v.Kind() {
	case reflect.Struct:
		return decodeStructFromDecoder(d, v)
	default:
		return fmt.Errorf("cannot decode container into %v", v.Kind())
	}
}