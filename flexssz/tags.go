package flexssz

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/gfx-labs/ssz"
	"github.com/holiman/uint256"
)

var (
	// Precalculated type to avoid reflection overhead
	uint256TypeTag = reflect.TypeOf(uint256.Int{})
)

// sszTag represents parsed SSZ struct tag information
type sszTag struct {
	Skip       bool   // "-" tag means skip this field
	FieldType  string // "uint8", "uint16", "uint32", "uint64", "bool", "vector", "list", "container", "string", "bitlist", "bitvector"
	IsVariable bool   // Whether this field is variable-size (strings, slices)
	MaxList    int    // For variable-size lists: ssz-max:"1024"
	Size       []int  // For fixed-size arrays: ssz-size:"32" or "8192,32" for multi-dimensional
}

// TypeInfo represents SSZ type information for any type (not just structs)
type TypeInfo struct {
	Type       ssz.TypeName  // SSZ type name
	FixedSize  int           // Size in bytes for fixed types, -1 for variable
	IsVariable bool          // Whether this type is variable-size

	// For basic types
	BasicType reflect.Type // The underlying Go type for basic types

	// For containers (structs)
	Fields []FieldInfo // Fields for container types

	// For lists and vectors
	ElementType *TypeInfo // Element type info for lists/vectors
	Length      int       // Fixed length for vectors, max length for lists (0 = unlimited)

	// For special types
	BitLength int     // Number of bits for bitvector/bitlist
	Tag       *sszTag // Original tag information
}

// FieldInfo represents information about a struct field
type FieldInfo struct {
	Index  int       // Field index in struct
	Name   string    // Field name
	Type   *TypeInfo // Type information for this field
	Offset int       // Offset in fixed part (-1 for variable fields)
}


// typeInfoCache caches parsed type information
var typeInfoCache = make(map[reflect.Type]*TypeInfo)
var typeInfoCacheMutex sync.RWMutex

// parseSSZTags parses SSZ-related struct tags
func parseSSZTags(field reflect.StructField) (*sszTag, error) {
	tag := &sszTag{}

	// Check for skip tag or explicit type
	sszTag := field.Tag.Get("ssz")
	if sszTag == "-" {
		tag.Skip = true
		return tag, nil
	} else if sszTag != "" {
		// If ssz tag has a value, use it as the field type
		tag.FieldType = sszTag
	}

	// Parse ssz-size tag for fixed-size arrays/slices
	if sizeStr := field.Tag.Get("ssz-size"); sizeStr != "" {
		// Handle multi-dimensional sizes like "8192,32"
		parts := strings.Split(sizeStr, ",")
		sizes := make([]int, len(parts))
		for i, part := range parts {
			size, err := strconv.Atoi(strings.TrimSpace(part))
			if err != nil {
				return nil, fmt.Errorf("invalid ssz-size value: %v", err)
			}
			sizes[i] = size
		}
		tag.Size = sizes

		// Don't auto-set field type for slices with ssz-size
		// They will be handled based on reflection
	}

	// Parse ssz-max tag for variable-size lists
	if maxStr := field.Tag.Get("ssz-max"); maxStr != "" {
		// Handle special case "?" which means no limit
		if maxStr == "?" {
			tag.MaxList = 0 // 0 means no limit in our implementation
		} else {
			max, err := strconv.Atoi(maxStr)
			if err != nil {
				return nil, fmt.Errorf("invalid ssz-max value: %v", err)
			}
			tag.MaxList = max
		}
		tag.IsVariable = true

		// Don't auto-set field type for slices with ssz-max
		// They will be handled based on reflection
	}

	// Auto-detect field type based on reflection if not specified
	if tag.FieldType == "" {
		tag.FieldType = detectFieldType(field.Type)
	}

	// Validate field type
	if err := validateFieldType(field, tag); err != nil {
		return nil, err
	}

	// Validate ssz-size and ssz-max usage
	if len(tag.Size) > 0 && tag.MaxList > 0 {
		return nil, fmt.Errorf("field %s: cannot use both ssz-size and ssz-max tags", field.Name)
	}

	// Validate ssz-size can only be used with arrays or slices
	if len(tag.Size) > 0 && field.Type.Kind() != reflect.Array && field.Type.Kind() != reflect.Slice {
		return nil, fmt.Errorf("field %s: ssz-size tag can only be used with array or slice types, got %v", field.Name, field.Type)
	}

	// Validate ssz-max can only be used with slices
	if tag.MaxList > 0 && field.Type.Kind() != reflect.Slice {
		return nil, fmt.Errorf("field %s: ssz-max tag can only be used with slice types, got %v", field.Name, field.Type)
	}

	// Validate that variable slices must have a limit
	// Note: MaxList == 0 after parsing "?" means no limit, which is valid
	if field.Type.Kind() == reflect.Slice && len(tag.Size) == 0 && tag.MaxList == 0 && field.Tag.Get("ssz-max") == "" {
		return nil, fmt.Errorf("field %s: slice types must have either ssz-size or ssz-max tag", field.Name)
	}

	// Validate multi-dimensional arrays
	if len(tag.Size) > 1 {
		// Check that we have nested slices/arrays
		t := field.Type
		for i, size := range tag.Size {
			if t.Kind() != reflect.Slice && t.Kind() != reflect.Array {
				return nil, fmt.Errorf("field %s: ssz-size has %d dimensions but type only has %d", field.Name, len(tag.Size), i)
			}

			// If it's an array, validate the size matches
			if t.Kind() == reflect.Array && t.Len() != size {
				return nil, fmt.Errorf("field %s: array size %d does not match ssz-size %d at dimension %d", field.Name, t.Len(), size, i)
			}

			t = t.Elem()
		}
	}

	// Determine if field is variable-size based on type
	// Note: slices with ssz-size are fixed-size, not variable
	if !tag.IsVariable && len(tag.Size) == 0 {
		tag.IsVariable = typeIsVariable(field.Type, tag)
	}

	return tag, nil
}

// detectFieldType determines the SSZ type based on reflection
func detectFieldType(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Uint8:
		return "uint8"
	case reflect.Uint16:
		return "uint16"
	case reflect.Uint32:
		return "uint32"
	case reflect.Uint64:
		return "uint64"
	case reflect.Bool:
		return "bool"
	case reflect.String:
		return "string"
	case reflect.Slice:
		// For slices, default to list
		return "list"
	case reflect.Array:
		// Check if it's a uint256.Int type (which is [4]uint64)
		if t == uint256TypeTag {
			// By default, treat as uint256 unless tag specifies otherwise
			return "uint256"
		}
		return "vector"
	case reflect.Struct:
		return "container"
	case reflect.Ptr:
		// For pointers, detect based on the element type
		return detectFieldType(t.Elem())
	default:
		return ""
	}
}

// typeIsVariable determines if a type is variable-size in SSZ.
// This considers both the type itself and any SSZ tag information.
func typeIsVariable(t reflect.Type, tag *sszTag) bool {
	// Tag explicitly marks it as variable
	if tag != nil && tag.IsVariable {
		return true
	}

	switch t.Kind() {
	case reflect.String:
		return true
	case reflect.Slice:
		// Slices with ssz-size are fixed-size
		if tag != nil && len(tag.Size) > 0 {
			return false
		}
		// Otherwise slices are variable-size
		return true
	case reflect.Array:
		// Arrays are fixed-size, but check if they contain variable elements
		// An array of variable-size elements is still fixed-size overall
		// but needs special handling during encoding
		return false
	case reflect.Struct:
		// Check if struct contains any variable-size fields
		return structHasVariableFields(t)
	case reflect.Ptr:
		// Pointers are variable if their element is variable
		return typeIsVariable(t.Elem(), tag)
	default:
		// Basic types are fixed
		return false
	}
}

// validateFieldType validates that the field type matches the SSZ tag
func validateFieldType(field reflect.StructField, tag *sszTag) error {
	t := field.Type

	switch tag.FieldType {
	case "uint8":
		if t.Kind() != reflect.Uint8 {
			return fmt.Errorf("field %s: ssz tag 'uint8' requires Go type uint8, got %v", field.Name, t)
		}
	case "uint16":
		if t.Kind() != reflect.Uint16 {
			return fmt.Errorf("field %s: ssz tag 'uint16' requires Go type uint16, got %v", field.Name, t)
		}
	case "uint32":
		if t.Kind() != reflect.Uint32 {
			return fmt.Errorf("field %s: ssz tag 'uint32' requires Go type uint32, got %v", field.Name, t)
		}
	case "uint64":
		if t.Kind() != reflect.Uint64 {
			return fmt.Errorf("field %s: ssz tag 'uint64' requires Go type uint64, got %v", field.Name, t)
		}
	case "bool":
		if t.Kind() != reflect.Bool {
			return fmt.Errorf("field %s: ssz tag 'bool' requires Go type bool, got %v", field.Name, t)
		}
	case "string":
		if t.Kind() != reflect.String {
			return fmt.Errorf("field %s: ssz tag 'string' requires Go type string, got %v", field.Name, t)
		}
	case "list":
		// list must be a slice type
		if t.Kind() != reflect.Slice {
			return fmt.Errorf("field %s: ssz tag 'list' requires slice type, got %v", field.Name, t)
		}
	case "vector":
		// vector must be an array type
		if t.Kind() != reflect.Array {
			return fmt.Errorf("field %s: ssz tag 'vector' requires array type, got %v", field.Name, t)
		}
	case "uint128", "uint256":
		// Allow both uint256.Int and *uint256.Int
		if t == uint256TypeTag {
			// Direct uint256.Int type
		} else if t.Kind() == reflect.Ptr && t.Elem() == uint256TypeTag {
			// Pointer to uint256.Int
		} else {
			return fmt.Errorf("field %s: ssz tag '%s' requires uint256.Int or *uint256.Int type, got %v", field.Name, tag.FieldType, t)
		}
	case "container":
		// container must be a struct type or pointer to struct
		if t.Kind() == reflect.Ptr {
			if t.Elem().Kind() != reflect.Struct {
				return fmt.Errorf("field %s: ssz tag 'container' requires struct or pointer to struct type, got %v", field.Name, t)
			}
		} else if t.Kind() != reflect.Struct {
			return fmt.Errorf("field %s: ssz tag 'container' requires struct or pointer to struct type, got %v", field.Name, t)
		}
	case "bitlist":
		// bitlist must be a []byte type
		if t.Kind() != reflect.Slice || t.Elem().Kind() != reflect.Uint8 {
			return fmt.Errorf("field %s: ssz tag 'bitlist' requires []byte type, got %v", field.Name, t)
		}
		// bitlist requires ssz-max tag
		if tag.MaxList == 0 && field.Tag.Get("ssz-max") == "" {
			return fmt.Errorf("field %s: bitlist requires ssz-max tag", field.Name)
		}
	case "bitvector":
		// bitvector must be a []byte type
		if t.Kind() != reflect.Slice || t.Elem().Kind() != reflect.Uint8 {
			return fmt.Errorf("field %s: ssz tag 'bitvector' requires []byte type, got %v", field.Name, t)
		}
		// bitvector requires ssz-size tag
		if len(tag.Size) == 0 {
			return fmt.Errorf("field %s: bitvector requires ssz-size tag", field.Name)
		}
	case "":
		// No explicit type specified, auto-detected
		if detectFieldType(t) == "" {
			return fmt.Errorf("field %s: unsupported type %v for SSZ encoding", field.Name, t)
		}
	default:
		// For other types like "container", we don't validate strictly
		// as they depend on the actual Go type
	}

	return nil
}


// structHasVariableFields checks if a struct type contains any variable-size fields
func structHasVariableFields(t reflect.Type) bool {
	if t.Kind() != reflect.Struct {
		return false
	}

	typeInfo, err := GetTypeInfo(t, nil)
	if err != nil {
		// If we can't parse, assume it's not variable-size
		return false
	}

	return typeInfo.IsVariable
}

// PrecacheStructSSZInfo precaches SSZ encoding information for a struct type
// This is useful to call in init() functions to validate struct tags early
// and improve performance by avoiding repeated parsing
func PrecacheStructSSZInfo(v interface{}) error {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Only cache top-level structs
	if t.Kind() != reflect.Struct {
		return fmt.Errorf("PrecacheStructSSZInfo can only be used with struct types, got %v", t.Kind())
	}

	// Check if already cached
	typeInfoCacheMutex.RLock()
	_, exists := typeInfoCache[t]
	typeInfoCacheMutex.RUnlock()

	if exists {
		return nil
	}

	// Parse and cache the struct
	info, err := parseTypeInfo(t, nil)
	if err != nil {
		return err
	}

	// Cache the result
	typeInfoCacheMutex.Lock()
	typeInfoCache[t] = info
	typeInfoCacheMutex.Unlock()

	return nil
}

// MustPrecacheStructSSZInfo precaches SSZ encoding information for a struct type
// and panics if there is an error. This is useful in init() functions or tests
// where you want to ensure struct tags are valid at startup.
func MustPrecacheStructSSZInfo(v interface{}) {
	if err := PrecacheStructSSZInfo(v); err != nil {
		panic(fmt.Sprintf("failed to precache SSZ info: %v", err))
	}
}

// GetTypeInfo returns type information for any Go type
func GetTypeInfo(t reflect.Type, tag *sszTag) (*TypeInfo, error) {
	// Only use cache when tag is nil (typically for top-level structs)
	if tag == nil {
		typeInfoCacheMutex.RLock()
		info, exists := typeInfoCache[t]
		typeInfoCacheMutex.RUnlock()
		
		if exists {
			return info, nil
		}
	}
	
	// Parse type info
	info, err := parseTypeInfo(t, tag)
	if err != nil {
		return nil, err
	}
	
	// Cache the result only when tag is nil
	if tag == nil {
		typeInfoCacheMutex.Lock()
		typeInfoCache[t] = info
		typeInfoCacheMutex.Unlock()
	}
	
	return info, nil
}

// parseTypeInfo parses type information for any Go type
// calculateIsVariable recursively calculates the IsVariable field for a TypeInfo
func calculateIsVariable(info *TypeInfo) {
	// Use the SSZ type methods to determine variability
	if info.Type.IsAlwaysFixed() {
		info.IsVariable = false
		return
	}
	
	if info.Type.IsAlwaysVariable() {
		info.IsVariable = true
		return
	}
	
	// For types that are sometimes variable, we need to check children
	if info.Type.IsSometimesVariable() {
		switch info.Type {
		case ssz.TypeVector:
			// Vectors are fixed if their elements are fixed
			if info.ElementType != nil {
				calculateIsVariable(info.ElementType)
				info.IsVariable = info.ElementType.IsVariable
			} else {
				info.IsVariable = false
			}
		case ssz.TypeContainer:
			// Containers are variable if any field is variable
			info.IsVariable = false
			for _, field := range info.Fields {
				calculateIsVariable(field.Type)
				if field.Type.IsVariable {
					info.IsVariable = true
					break
				}
			}
		default:
			// For other sometimes-variable types, default to variable
			info.IsVariable = true
		}
	}
}

func parseTypeInfo(t reflect.Type, tag *sszTag) (*TypeInfo, error) {
	info := &TypeInfo{
		Tag: tag,
	}

	// Handle pointer types by dereferencing
	if t.Kind() == reflect.Ptr {
		// Get info for the element type
		elemInfo, err := GetTypeInfo(t.Elem(), tag)
		if err != nil {
			return nil, err
		}
		// Pointers have the same characteristics as their element
		return elemInfo, nil
	}

	switch t.Kind() {
	case reflect.Uint8:
		info.Type = ssz.TypeUint8
		info.BasicType = t
		info.FixedSize = 1

	case reflect.Uint16:
		info.Type = ssz.TypeUint16
		info.BasicType = t
		info.FixedSize = 2

	case reflect.Uint32:
		info.Type = ssz.TypeUint32
		info.BasicType = t
		info.FixedSize = 4

	case reflect.Uint64:
		info.Type = ssz.TypeUint64
		info.BasicType = t
		info.FixedSize = 8

	case reflect.Bool:
		info.Type = ssz.TypeBoolean
		info.BasicType = t
		info.FixedSize = 1

	case reflect.String:
		info.Type = ssz.TypeList  // String is represented as a list of bytes in SSZ
		info.FixedSize = -1
		// String is like a list of bytes
		info.ElementType = &TypeInfo{
			Type:      ssz.TypeUint8,
			BasicType: reflect.TypeOf(byte(0)),
			FixedSize: 1,
		}

	case reflect.Array:
		if t == uint256TypeTag {
			// Special case for uint256.Int
			info.BasicType = t
			if tag != nil && tag.FieldType == "uint128" {
				info.Type = ssz.TypeUint128
				info.FixedSize = 16
			} else {
				info.Type = ssz.TypeUint256
				info.FixedSize = 32
			}
		} else if tag != nil && tag.FieldType == "bitvector" {
			// Bitvector
			info.Type = ssz.TypeBitVector
			if len(tag.Size) > 0 {
				info.BitLength = tag.Size[0]
				info.FixedSize = (tag.Size[0] + 7) / 8
			} else {
				return nil, fmt.Errorf("bitvector requires ssz-size tag")
			}
		} else {
			// Regular array (vector)
			info.Type = ssz.TypeVector
			info.Length = t.Len()

			// Get element type info
			elemInfo, err := GetTypeInfo(t.Elem(), nil)
			if err != nil {
				return nil, err
			}
			info.ElementType = elemInfo

			// Calculate fixed size
			if elemInfo.FixedSize > 0 {
				info.FixedSize = info.Length * elemInfo.FixedSize
			} else {
				// Array of variable-size elements
				info.FixedSize = info.Length * 4 // Each element needs an offset
			}
		}

	case reflect.Slice:
		if tag != nil && len(tag.Size) > 0 {
			// Fixed-size slice (treated as vector)
			info.Type = ssz.TypeVector
			info.Length = tag.Size[0]

			// Get element type info
			elemTag := &sszTag{}
			if len(tag.Size) > 1 {
				elemTag.Size = tag.Size[1:]
			}
			elemInfo, err := GetTypeInfo(t.Elem(), elemTag)
			if err != nil {
				return nil, err
			}
			info.ElementType = elemInfo

			// Calculate fixed size
			if t.Elem().Kind() == reflect.Uint8 && tag.FieldType == "bitvector" {
				info.Type = ssz.TypeBitVector
				info.BitLength = tag.Size[0]
				info.FixedSize = (tag.Size[0] + 7) / 8
			} else if elemInfo.FixedSize > 0 {
				info.FixedSize = info.Length * elemInfo.FixedSize
			} else {
				// Fixed-size array of variable elements
				info.FixedSize = info.Length * 4
			}
		} else {
			// Variable-size slice (list)
			if tag != nil && tag.FieldType == "bitlist" {
				info.Type = ssz.TypeBitList
				info.BitLength = tag.MaxList
			} else {
				info.Type = ssz.TypeList
			}
			info.FixedSize = -1
			if tag != nil {
				info.Length = tag.MaxList // Max length
			}

			// Get element type info
			elemInfo, err := GetTypeInfo(t.Elem(), nil)
			if err != nil {
				return nil, err
			}
			info.ElementType = elemInfo
		}

	case reflect.Struct:
		info.Type = ssz.TypeContainer

		// Parse struct fields
		fields := make([]FieldInfo, 0, t.NumField())
		fixedOffset := 0
		hasVariable := false

		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)

			// Parse field tags
			fieldTag, err := parseSSZTags(field)
			if err != nil {
				return nil, err
			}

			// Skip ignored fields
			if fieldTag.Skip || !field.IsExported() {
				continue
			}

			// Get field type info
			fieldTypeInfo, err := GetTypeInfo(field.Type, fieldTag)
			if err != nil {
				return nil, err
			}

			fieldInfo := FieldInfo{
				Index: i,
				Name:  field.Name,
				Type:  fieldTypeInfo,
			}

			// Calculate offset
			if fieldTypeInfo.IsVariable {
				fieldInfo.Offset = -1
				fixedOffset += 4 // Offset pointer
				hasVariable = true
			} else {
				fieldInfo.Offset = fixedOffset
				fixedOffset += fieldTypeInfo.FixedSize
			}

			fields = append(fields, fieldInfo)
		}

		info.Fields = fields
		if hasVariable {
			info.FixedSize = -1
		} else {
			info.FixedSize = fixedOffset
		}

	default:
		return nil, fmt.Errorf("unsupported type for SSZ: %v", t)
	}

	// After fully populating TypeInfo, calculate IsVariable recursively
	calculateIsVariable(info)

	return info, nil
}
