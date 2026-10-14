package ssztest

import (
	"encoding/binary"
	"fmt"

	"github.com/gfx-labs/cuessz"
)

// ReferenceImplementation is a simple reference implementation for testing
// It implements basic SSZ serialization from scratch
type ReferenceImplementation struct{}

func NewReferenceImplementation() *ReferenceImplementation {
	return &ReferenceImplementation{}
}

func (r *ReferenceImplementation) Name() string {
	return "reference"
}

func (r *ReferenceImplementation) Marshal(value interface{}, def cuessz.Def) ([]byte, error) {
	return r.marshal(value, def)
}

func (r *ReferenceImplementation) Unmarshal(data []byte, def cuessz.Def) (interface{}, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *ReferenceImplementation) HashTreeRoot(value interface{}, def cuessz.Def) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *ReferenceImplementation) marshal(value interface{}, def cuessz.Def) ([]byte, error) {
	switch def.Type {
	case cuessz.TypeBoolean:
		b := value.(bool)
		if b {
			return []byte{1}, nil
		}
		return []byte{0}, nil

	case cuessz.TypeUint8:
		return []byte{value.(uint8)}, nil

	case cuessz.TypeUint16:
		v := value.(uint16)
		buf := make([]byte, 2)
		binary.LittleEndian.PutUint16(buf, v)
		return buf, nil

	case cuessz.TypeUint32:
		v := value.(uint32)
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, v)
		return buf, nil

	case cuessz.TypeUint64:
		v := value.(uint64)
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, v)
		return buf, nil

	case cuessz.TypeUint128, cuessz.TypeUint256:
		return value.([]byte), nil

	case cuessz.TypeBitVector:
		return value.([]byte), nil

	case cuessz.TypeBitList:
		return value.([]byte), nil

	case cuessz.TypeVector:
		if len(def.Children) == 0 {
			return nil, fmt.Errorf("vector has no element type")
		}

		elemDef := def.Children[0].Def
		elements := value.([]interface{})

		// Special case: vector of bytes
		if elemDef.Type == cuessz.TypeUint8 {
			result := make([]byte, 0, len(elements))
			for _, elem := range elements {
				result = append(result, elem.(uint8))
			}
			return result, nil
		}

		// Check if element type is fixed size
		isFixed := r.isFixedSize(elemDef)

		if isFixed {
			// Fixed-size elements: serialize sequentially
			var result []byte
			for _, elem := range elements {
				data, err := r.marshal(elem, elemDef)
				if err != nil {
					return nil, err
				}
				result = append(result, data...)
			}
			return result, nil
		} else {
			// Variable-size elements: offsets + data
			numElements := len(elements)
			offsetsSize := numElements * 4

			// Serialize all elements
			elementData := make([][]byte, numElements)
			for i, elem := range elements {
				data, err := r.marshal(elem, elemDef)
				if err != nil {
					return nil, err
				}
				elementData[i] = data
			}

			// Calculate total size
			totalSize := offsetsSize
			for _, data := range elementData {
				totalSize += len(data)
			}

			result := make([]byte, totalSize)

			// Write offsets
			currentOffset := uint32(offsetsSize)
			for i := 0; i < numElements; i++ {
				binary.LittleEndian.PutUint32(result[i*4:(i+1)*4], currentOffset)
				currentOffset += uint32(len(elementData[i]))
			}

			// Write data
			pos := offsetsSize
			for _, data := range elementData {
				copy(result[pos:], data)
				pos += len(data)
			}

			return result, nil
		}

	case cuessz.TypeList:
		if len(def.Children) == 0 {
			return nil, fmt.Errorf("list has no element type")
		}

		elemDef := def.Children[0].Def
		elements := value.([]interface{})

		// Special case: list of bytes
		if elemDef.Type == cuessz.TypeUint8 {
			result := make([]byte, 0, len(elements))
			for _, elem := range elements {
				result = append(result, elem.(uint8))
			}
			return result, nil
		}

		// Check if element type is fixed size
		isFixed := r.isFixedSize(elemDef)

		if isFixed {
			// Fixed-size elements: serialize sequentially
			var result []byte
			for _, elem := range elements {
				data, err := r.marshal(elem, elemDef)
				if err != nil {
					return nil, err
				}
				result = append(result, data...)
			}
			return result, nil
		} else {
			// Variable-size elements: same as variable vector
			numElements := len(elements)
			offsetsSize := numElements * 4

			elementData := make([][]byte, numElements)
			for i, elem := range elements {
				data, err := r.marshal(elem, elemDef)
				if err != nil {
					return nil, err
				}
				elementData[i] = data
			}

			totalSize := offsetsSize
			for _, data := range elementData {
				totalSize += len(data)
			}

			result := make([]byte, totalSize)

			currentOffset := uint32(offsetsSize)
			for i := 0; i < numElements; i++ {
				binary.LittleEndian.PutUint32(result[i*4:(i+1)*4], currentOffset)
				currentOffset += uint32(len(elementData[i]))
			}

			pos := offsetsSize
			for _, data := range elementData {
				copy(result[pos:], data)
				pos += len(data)
			}

			return result, nil
		}

	case cuessz.TypeContainer, cuessz.TypeProgressiveContainer:
		container := value.(map[string]interface{})

		// Separate fixed and variable fields
		var fixedFields []cuessz.Field
		var variableFields []cuessz.Field

		for i, field := range def.Children {
			// For progressive containers, skip inactive fields
			if def.Type == cuessz.TypeProgressiveContainer {
				if i >= len(def.ActiveFields) || def.ActiveFields[i] == 0 {
					continue
				}
			}

			if r.isFixedSize(field.Def) {
				fixedFields = append(fixedFields, field)
			} else {
				variableFields = append(variableFields, field)
			}
		}

		// Calculate fixed part size
		fixedSize := 0
		for _, field := range fixedFields {
			size, _ := r.calculateFieldSize(field.Def)
			fixedSize += size
		}
		fixedSize += len(variableFields) * 4 // offsets

		// Serialize fixed fields
		fixedPart := make([]byte, fixedSize)
		pos := 0
		for _, field := range fixedFields {
			val, ok := container[field.Name]
			if !ok {
				return nil, fmt.Errorf("missing field: %s", field.Name)
			}
			data, err := r.marshal(val, field.Def)
			if err != nil {
				return nil, err
			}
			copy(fixedPart[pos:], data)
			size, _ := r.calculateFieldSize(field.Def)
			pos += size
		}

		// Serialize variable fields
		variableData := make([][]byte, len(variableFields))
		for i, field := range variableFields {
			val, ok := container[field.Name]
			if !ok {
				return nil, fmt.Errorf("missing field: %s", field.Name)
			}
			data, err := r.marshal(val, field.Def)
			if err != nil {
				return nil, err
			}
			variableData[i] = data
		}

		// Write offsets
		currentOffset := uint32(fixedSize)
		offsetPos := len(fixedFields) * r.getFixedFieldsSize(fixedFields)
		for i := 0; i < len(variableFields); i++ {
			binary.LittleEndian.PutUint32(fixedPart[offsetPos:], currentOffset)
			offsetPos += 4
			currentOffset += uint32(len(variableData[i]))
		}

		// Combine fixed and variable parts
		result := fixedPart
		for _, data := range variableData {
			result = append(result, data...)
		}

		return result, nil

	default:
		return nil, fmt.Errorf("unsupported type: %s", def.Type)
	}
}

func (r *ReferenceImplementation) isFixedSize(def cuessz.Def) bool {
	switch def.Type {
	case cuessz.TypeBoolean, cuessz.TypeUint8, cuessz.TypeUint16, cuessz.TypeUint32,
		cuessz.TypeUint64, cuessz.TypeUint128, cuessz.TypeUint256, cuessz.TypeBitVector:
		return true
	case cuessz.TypeVector:
		if len(def.Children) == 0 {
			return false
		}
		return r.isFixedSize(def.Children[0].Def)
	case cuessz.TypeContainer:
		for _, child := range def.Children {
			if !r.isFixedSize(child.Def) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func (r *ReferenceImplementation) calculateFieldSize(def cuessz.Def) (int, error) {
	switch def.Type {
	case cuessz.TypeBoolean, cuessz.TypeUint8:
		return 1, nil
	case cuessz.TypeUint16:
		return 2, nil
	case cuessz.TypeUint32:
		return 4, nil
	case cuessz.TypeUint64:
		return 8, nil
	case cuessz.TypeUint128:
		return 16, nil
	case cuessz.TypeUint256:
		return 32, nil
	case cuessz.TypeBitVector:
		return int((def.Size + 7) / 8), nil
	case cuessz.TypeVector:
		if len(def.Children) == 0 {
			return 0, fmt.Errorf("vector has no element type")
		}
		elemSize, err := r.calculateFieldSize(def.Children[0].Def)
		if err != nil {
			return 0, err
		}
		return elemSize * int(def.Size), nil
	case cuessz.TypeContainer:
		total := 0
		for _, child := range def.Children {
			size, err := r.calculateFieldSize(child.Def)
			if err != nil {
				return 0, err
			}
			total += size
		}
		return total, nil
	default:
		return 0, fmt.Errorf("not fixed size: %s", def.Type)
	}
}

func (r *ReferenceImplementation) getFixedFieldsSize(fields []cuessz.Field) int {
	total := 0
	for _, field := range fields {
		size, _ := r.calculateFieldSize(field.Def)
		total += size
	}
	return total
}
