package flexssz

import (
	"fmt"
	"reflect"

	"github.com/gfx-labs/ssz"
)

// decodeFixedField decodes a fixed-size field
func decodeFixedField(d *Decoder, v reflect.Value, fieldInfo *FieldInfo) error {
	// Handle pointer types - but not for uint256 which has special handling
	if v.Kind() == reflect.Ptr && v.Type().Elem() != uint256Type {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		// Decode into the dereferenced value
		return decodeFixedField(d, v.Elem(), fieldInfo)
	}

	// Switch on SSZ type
	switch fieldInfo.Type.Type {
	case ssz.TypeUint8:
		return decodeUint8(d, v)
	case ssz.TypeUint16:
		return decodeUint16(d, v)
	case ssz.TypeUint32:
		return decodeUint32(d, v)
	case ssz.TypeUint64:
		return decodeUint64(d, v)
	case ssz.TypeUint128:
		return decodeUint128(d, v)
	case ssz.TypeUint256:
		return decodeUint256(d, v)
	case ssz.TypeBoolean:
		return decodeBoolean(d, v)
	case ssz.TypeBitVector:
		return decodeBitVector(d, v, fieldInfo)
	case ssz.TypeVector:
		return decodeVector(d, v, fieldInfo)
	case ssz.TypeContainer:
		return decodeContainer(d, v, fieldInfo)
	default:
		return fmt.Errorf("unsupported SSZ type for fixed field: %v", fieldInfo.Type.Type)
	}
}

// decodeUint8 decodes a uint8 value
func decodeUint8(d *Decoder, v reflect.Value) error {
	val, err := d.ReadUint8()
	if err != nil {
		return err
	}

	switch v.Kind() {
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		v.SetUint(uint64(val))
		return nil
	default:
		return fmt.Errorf("cannot decode uint8 into %v", v.Kind())
	}
}

// decodeUint16 decodes a uint16 value
func decodeUint16(d *Decoder, v reflect.Value) error {
	val, err := d.ReadUint16()
	if err != nil {
		return err
	}

	switch v.Kind() {
	case reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		v.SetUint(uint64(val))
		return nil
	default:
		return fmt.Errorf("cannot decode uint16 into %v", v.Kind())
	}
}

// decodeUint32 decodes a uint32 value
func decodeUint32(d *Decoder, v reflect.Value) error {
	val, err := d.ReadUint32()
	if err != nil {
		return err
	}

	switch v.Kind() {
	case reflect.Uint32, reflect.Uint64, reflect.Uint:
		v.SetUint(uint64(val))
		return nil
	default:
		return fmt.Errorf("cannot decode uint32 into %v", v.Kind())
	}
}

// decodeUint64 decodes a uint64 value
func decodeUint64(d *Decoder, v reflect.Value) error {
	val, err := d.ReadUint64()
	if err != nil {
		return err
	}

	switch v.Kind() {
	case reflect.Uint64, reflect.Uint:
		v.SetUint(val)
		return nil
	default:
		return fmt.Errorf("cannot decode uint64 into %v", v.Kind())
	}
}

// decodeUint128 decodes a uint128 value
func decodeUint128(d *Decoder, v reflect.Value) error {
	val, err := d.ReadUint128()
	if err != nil {
		return err
	}

	// Check if it's a uint256.Int type
	if v.Type() == uint256Type {
		v.Set(reflect.ValueOf(*val))
		return nil
	}

	// Check if it's a pointer to uint256.Int
	if v.Kind() == reflect.Ptr && v.Type().Elem() == uint256Type {
		if v.IsNil() {
			v.Set(reflect.New(uint256Type))
		}
		v.Elem().Set(reflect.ValueOf(*val))
		return nil
	}

	return fmt.Errorf("cannot decode uint128 into %v (expected uint256.Int or *uint256.Int)", v.Type())
}

// decodeUint256 decodes a uint256 value
func decodeUint256(d *Decoder, v reflect.Value) error {
	val, err := d.ReadUint256()
	if err != nil {
		return err
	}

	// Check if it's a uint256.Int type
	if v.Type() == uint256Type {
		v.Set(reflect.ValueOf(*val))
		return nil
	}

	// Check if it's a pointer to uint256.Int
	if v.Kind() == reflect.Ptr && v.Type().Elem() == uint256Type {
		if v.IsNil() {
			v.Set(reflect.New(uint256Type))
		}
		v.Elem().Set(reflect.ValueOf(*val))
		return nil
	}

	return fmt.Errorf("cannot decode uint256 into %v (expected uint256.Int or *uint256.Int)", v.Type())
}

// decodeBoolean decodes a boolean value
func decodeBoolean(d *Decoder, v reflect.Value) error {
	val, err := d.ReadBool()
	if err != nil {
		return err
	}

	switch v.Kind() {
	case reflect.Bool:
		v.SetBool(val)
		return nil
	default:
		return fmt.Errorf("cannot decode boolean into %v", v.Kind())
	}
}

// decodeBitVector decodes a bitvector
func decodeBitVector(d *Decoder, v reflect.Value, fieldInfo *FieldInfo) error {
	if fieldInfo.Type.BitLength == 0 {
		return fmt.Errorf("bitvector must have BitLength set")
	}

	bytesToRead := (fieldInfo.Type.BitLength + 7) / 8
	bytes, err := d.ReadN(bytesToRead)
	if err != nil {
		return err
	}

	// Decode bitvector (validate no extra bits)
	decoded, err := DecodeBitVector(bytes, fieldInfo.Type.BitLength)
	if err != nil {
		return fmt.Errorf("error decoding bitvector: %w", err)
	}

	switch v.Kind() {
	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			v.SetBytes(decoded)
			return nil
		}
		return fmt.Errorf("cannot decode bitvector into slice of %v", v.Type().Elem())
	default:
		return fmt.Errorf("cannot decode bitvector into %v", v.Kind())
	}
}

// decodeVector decodes a fixed-size vector
func decodeVector(d *Decoder, v reflect.Value, fieldInfo *FieldInfo) error {
	length := fieldInfo.Type.Length
	if length == 0 {
		return fmt.Errorf("vector must have Length set")
	}

	elemType := fieldInfo.Type.ElementType
	if elemType == nil {
		return fmt.Errorf("vector must have ElementType set")
	}

	switch v.Kind() {
	case reflect.Array:
		if v.Len() != length {
			return fmt.Errorf("array length %d does not match vector length %d", v.Len(), length)
		}
		// Special case for byte arrays
		if v.Type().Elem().Kind() == reflect.Uint8 && elemType.Type == ssz.TypeUint8 {
			bytes, err := d.ReadN(length)
			if err != nil {
				return err
			}
			for i := 0; i < length; i++ {
				v.Index(i).SetUint(uint64(bytes[i]))
			}
			return nil
		}
		// Decode each element
		for i := 0; i < length; i++ {
			elemFieldInfo := &FieldInfo{
				Type: elemType,
				Name: fmt.Sprintf("%s[%d]", fieldInfo.Name, i),
			}
			if err := decodeFixedField(d, v.Index(i), elemFieldInfo); err != nil {
				return err
			}
		}
		return nil

	case reflect.Slice:
		// Create slice with proper length
		v.Set(reflect.MakeSlice(v.Type(), length, length))

		// Special case for byte slices
		if v.Type().Elem().Kind() == reflect.Uint8 && elemType.Type == ssz.TypeUint8 {
			bytes, err := d.ReadN(length)
			if err != nil {
				return err
			}
			v.SetBytes(bytes)
			return nil
		}

		// Decode each element
		for i := 0; i < length; i++ {
			elemFieldInfo := &FieldInfo{
				Type: elemType,
				Name: fmt.Sprintf("%s[%d]", fieldInfo.Name, i),
			}
			if err := decodeFixedField(d, v.Index(i), elemFieldInfo); err != nil {
				return err
			}
		}
		return nil

	default:
		return fmt.Errorf("cannot decode vector into %v", v.Kind())
	}
}

// decodeContainer decodes a container (struct)
func decodeContainer(d *Decoder, v reflect.Value, fieldInfo *FieldInfo) error {
	switch v.Kind() {
	case reflect.Struct:
		return decodeStructFromDecoder(d, v)
	default:
		return fmt.Errorf("cannot decode container into %v", v.Kind())
	}
}

