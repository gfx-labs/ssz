package flexssz

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/holiman/uint256"
)

var (
	// Precalculated type to avoid reflection overhead
	uint256TypeTag = reflect.TypeOf(uint256.Int{})
)

// sszTag represents parsed SSZ struct tag information
type sszTag struct {
	Skip       bool     // "-" tag means skip this field
	FieldType  string   // "uint8", "uint16", "uint32", "uint64", "bool", "vector", "list", "container", "string", "bitlist", "bitvector"
	IsVariable bool     // Whether this field is variable-size (strings, slices)
	MaxList    int      // For variable-size lists: ssz-max:"1024"
	Size       []int    // For fixed-size arrays: ssz-size:"32" or "8192,32" for multi-dimensional
}

// structSSZInfo holds precached SSZ encoding information for a struct
type structSSZInfo struct {
	Fields []fieldSSZInfo
}

// fieldSSZInfo holds SSZ encoding information for a single field
type fieldSSZInfo struct {
	Index     int
	Name      string
	Type      reflect.Type
	Tag        *sszTag
	IsVariable bool
}

// structInfoCache caches parsed struct information
var structInfoCache = make(map[reflect.Type]*structSSZInfo)
var structInfoCacheMutex sync.RWMutex

// parseSSZTags parses SSZ-related struct tags
func parseSSZTags(field reflect.StructField) (*sszTag, error) {
	tag := &sszTag{}

	// Check for skip tag
	sszTag := field.Tag.Get("ssz")
	if sszTag == "-" {
		tag.Skip = true
		return tag, nil
	}

	// Parse SSZ type if specified
	if sszTag != "" {
		tag.FieldType = sszTag
		// bitlist is always variable-size
		if sszTag == "bitlist" {
			tag.IsVariable = true
		}
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

	// Recursively validate nested types
	if err := validateNestedTypes(field.Type, field.Name); err != nil {
		return nil, err
	}

	// Determine if field is variable-size based on type
	if !tag.IsVariable {
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

// getStructSSZInfo returns cached SSZ encoding information for a struct type
func getStructSSZInfo(t reflect.Type) (*structSSZInfo, error) {
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct type, got %v", t.Kind())
	}

	// Check cache first
	structInfoCacheMutex.RLock()
	info, exists := structInfoCache[t]
	structInfoCacheMutex.RUnlock()

	if exists {
		return info, nil
	}

	// Parse struct if not cached
	info, err := parseStructSSZInfo(t)
	if err != nil {
		return nil, err
	}

	// Cache the result
	structInfoCacheMutex.Lock()
	structInfoCache[t] = info
	structInfoCacheMutex.Unlock()

	return info, nil
}

// parseStructSSZInfo parses SSZ encoding information for a struct
func parseStructSSZInfo(t reflect.Type) (*structSSZInfo, error) {
	info := &structSSZInfo{
		Fields: make([]fieldSSZInfo, 0, t.NumField()),
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Parse tags
		tag, err := parseSSZTags(field)
		if err != nil {
			return nil, err
		}

		// Skip ignored fields
		if tag.Skip || !field.IsExported() {
			continue
		}

		fieldInfo := fieldSSZInfo{
			Index:     i,
			Name:      field.Name,
			Type:      field.Type,
			Tag:       tag,
			IsVariable: tag.IsVariable || typeIsVariable(field.Type, tag),
		}

		info.Fields = append(info.Fields, fieldInfo)
	}

	return info, nil
}

// structHasVariableFields checks if a struct type contains any variable-size fields
func structHasVariableFields(t reflect.Type) bool {
	if t.Kind() != reflect.Struct {
		return false
	}

	info, err := getStructSSZInfo(t)
	if err != nil {
		// If we can't parse, assume it's not variable-size
		return false
	}

	for _, field := range info.Fields {
		if field.IsVariable {
			return true
		}
	}

	return false
}

// PrecacheStructSSZInfo precaches SSZ encoding information for a struct type
// This is useful to call in init() functions to validate struct tags early
// and improve performance by avoiding repeated parsing
func PrecacheStructSSZInfo(v interface{}) error {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	_, err := getStructSSZInfo(t)
	return err
}

// validateNestedTypes recursively validates nested types
func validateNestedTypes(t reflect.Type, fieldPath string) error {
	switch t.Kind() {
	case reflect.Slice:
		// Check element type
		elemType := t.Elem()
		
		// Multi-dimensional slices are allowed with ssz-size tags
		// The validation happens elsewhere
		
		// Recursively check element type
		return validateNestedTypes(elemType, fieldPath+"[]")
		
	case reflect.Array:
		// Check element type
		return validateNestedTypes(t.Elem(), fieldPath+"[...]")
		
	case reflect.Ptr:
		// Check pointed type
		return validateNestedTypes(t.Elem(), fieldPath+"*")
		
	case reflect.Struct:
		// For structs, we don't need to validate here as they'll be validated
		// when getStructSSZInfo is called on them
		return nil
		
	default:
		// Basic types are fine
		return nil
	}
}

