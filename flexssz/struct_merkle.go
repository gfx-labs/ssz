package flexssz

import (
	"encoding/binary"
	"fmt"
	"reflect"

	"github.com/gfx-labs/ssz"
	"github.com/gfx-labs/ssz/merkle_tree"
	"github.com/holiman/uint256"
)

const BYTES_PER_CHUNK = 32

// HashTreeRootStruct calculates the merkle root of a struct
func HashTreeRootStruct(v any) ([32]byte, error) {
	rv := reflect.ValueOf(v)
	
	// Handle pointer by dereferencing
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return [32]byte{}, fmt.Errorf("cannot hash nil pointer")
		}
		rv = rv.Elem()
	}
	
	// Must be a struct
	if rv.Kind() != reflect.Struct {
		return [32]byte{}, fmt.Errorf("HashTreeRootStruct requires a struct, got %v", rv.Kind())
	}
	
	// Get type info
	typeInfo, err := GetTypeInfo(rv.Type(), nil)
	if err != nil {
		return [32]byte{}, fmt.Errorf("error getting type info: %w", err)
	}
	
	if typeInfo.Type != ssz.TypeContainer {
		return [32]byte{}, fmt.Errorf("expected container type, got %v", typeInfo.Type)
	}
	
	// Calculate hash tree root
	return hashTreeRoot(rv, typeInfo)
}

// hashTreeRoot implements the recursive hash_tree_root function from the SSZ spec
func hashTreeRoot(v reflect.Value, typeInfo *TypeInfo) ([32]byte, error) {
	// Handle pointer types
	if v.Kind() == reflect.Ptr && v.Type().Elem() != uint256Type {
		if v.IsNil() {
			// For nil pointers, return zero hash
			return [32]byte{}, nil
		}
		return hashTreeRoot(v.Elem(), typeInfo)
	}
	
	switch typeInfo.Type {
	case ssz.TypeUint8, ssz.TypeUint16, ssz.TypeUint32, ssz.TypeUint64, ssz.TypeUint128, ssz.TypeUint256, ssz.TypeBoolean:
		// Basic types: merkleize(pack(value))
		chunks := packBasicValue(v, typeInfo)
		return merkleize(chunks, 0)
		
	case ssz.TypeBitVector:
		// Bitvectors: merkleize(pack_bits(value), limit=chunk_count(type))
		if v.Kind() != reflect.Slice || v.Type().Elem().Kind() != reflect.Uint8 {
			return [32]byte{}, fmt.Errorf("invalid type for bitvector: %v", v.Type())
		}
		chunks := packBytes(v.Bytes())
		limit := chunkCount(typeInfo)
		return merkleize(chunks, limit)
		
	case ssz.TypeBitList:
		// Bitlists: mix_in_length(merkleize(pack_bits(value), limit=chunk_count(type)), len(value))
		if v.Kind() != reflect.Slice || v.Type().Elem().Kind() != reflect.Uint8 {
			return [32]byte{}, fmt.Errorf("invalid type for bitlist: %v", v.Type())
		}
		return merkle_tree.BitlistRootWithLimit(v.Bytes(), uint64(typeInfo.BitLength))
		
	case ssz.TypeVector:
		// Vectors have different handling based on element type
		return hashTreeRootVector(v, typeInfo)
		
	case ssz.TypeList:
		// Lists have different handling based on element type
		return hashTreeRootList(v, typeInfo)
		
	case ssz.TypeContainer:
		// Containers: merkleize([hash_tree_root(element) for element in value])
		return hashTreeRootContainer(v, typeInfo)
		
	default:
		return [32]byte{}, fmt.Errorf("unsupported SSZ type for merkle root: %v", typeInfo.Type)
	}
}

// packBasicValue packs a single basic value into chunks
func packBasicValue(v reflect.Value, typeInfo *TypeInfo) [][32]byte {
	var chunk [32]byte
	
	switch typeInfo.Type {
	case ssz.TypeUint8:
		chunk[0] = uint8(v.Uint())
	case ssz.TypeUint16:
		binary.LittleEndian.PutUint16(chunk[:2], uint16(v.Uint()))
	case ssz.TypeUint32:
		binary.LittleEndian.PutUint32(chunk[:4], uint32(v.Uint()))
	case ssz.TypeUint64:
		binary.LittleEndian.PutUint64(chunk[:8], v.Uint())
	case ssz.TypeUint128, ssz.TypeUint256:
		if v.Type() == uint256Type {
			uint256Val := v.Interface().(uint256.Int)
			uint256Val.WriteToSlice(chunk[:])
		} else if v.Kind() == reflect.Ptr && v.Type().Elem() == uint256Type {
			if !v.IsNil() {
				uint256Val := v.Elem().Interface().(uint256.Int)
				uint256Val.WriteToSlice(chunk[:])
			}
		}
		if typeInfo.Type == ssz.TypeUint128 {
			// For uint128, zero out bytes 16-31
			for i := 16; i < 32; i++ {
				chunk[i] = 0
			}
		}
	case ssz.TypeBoolean:
		if v.Bool() {
			chunk[0] = 1
		}
	}
	
	return [][32]byte{chunk}
}

// packBytes packs bytes into chunks
func packBytes(data []byte) [][32]byte {
	// Calculate number of chunks needed
	numChunks := (len(data) + BYTES_PER_CHUNK - 1) / BYTES_PER_CHUNK
	if numChunks == 0 {
		numChunks = 1 // At least one chunk
	}
	
	chunks := make([][32]byte, numChunks)
	for i := 0; i < len(data); i++ {
		chunks[i/BYTES_PER_CHUNK][i%BYTES_PER_CHUNK] = data[i]
	}
	
	return chunks
}

// packBasicVector packs a vector of basic types into chunks
func packBasicVector(v reflect.Value, length int, elemType *TypeInfo) [][32]byte {
	var data []byte
	
	switch elemType.Type {
	case ssz.TypeUint8:
		data = make([]byte, length)
		for i := 0; i < length && i < v.Len(); i++ {
			data[i] = uint8(v.Index(i).Uint())
		}
	case ssz.TypeUint16:
		data = make([]byte, length*2)
		for i := 0; i < length && i < v.Len(); i++ {
			binary.LittleEndian.PutUint16(data[i*2:], uint16(v.Index(i).Uint()))
		}
	case ssz.TypeUint32:
		data = make([]byte, length*4)
		for i := 0; i < length && i < v.Len(); i++ {
			binary.LittleEndian.PutUint32(data[i*4:], uint32(v.Index(i).Uint()))
		}
	case ssz.TypeUint64:
		data = make([]byte, length*8)
		for i := 0; i < length && i < v.Len(); i++ {
			binary.LittleEndian.PutUint64(data[i*8:], v.Index(i).Uint())
		}
	case ssz.TypeBoolean:
		data = make([]byte, length)
		for i := 0; i < length && i < v.Len(); i++ {
			if v.Index(i).Bool() {
				data[i] = 1
			}
		}
	}
	
	return packBytes(data)
}

// merkleize implements the merkleize function from the SSZ spec
func merkleize(chunks [][32]byte, limit uint64) ([32]byte, error) {
	// MerkleizeVector has a bug with no limit, so we'll use a different approach
	
	if len(chunks) == 0 {
		// No chunks - return zero hash at appropriate depth
		if limit == 0 {
			return [32]byte{}, nil
		}
		depth := merkle_tree.GetDepth(limit)
		return merkle_tree.ZeroHashes[depth], nil
	}
	
	// Convert chunks to flat bytes
	flatBytes := make([]byte, len(chunks)*32)
	for i, chunk := range chunks {
		copy(flatBytes[i*32:(i+1)*32], chunk[:])
	}
	
	// Use ComputeMerkleRoot which works correctly
	output := make([]byte, 32)
	
	if limit > 0 {
		// With limit, use ComputeMerkleRootFromLevel
		err := merkle_tree.ComputeMerkleRootFromLevel(flatBytes, output, limit*32, 0)
		if err != nil {
			return [32]byte{}, err
		}
	} else {
		// No limit, use regular ComputeMerkleRoot
		err := merkle_tree.ComputeMerkleRoot(flatBytes, output)
		if err != nil {
			return [32]byte{}, err
		}
	}
	
	var result [32]byte
	copy(result[:], output)
	return result, nil
}

// mixInLength implements mix_in_length from the SSZ spec
func mixInLength(root [32]byte, length uint64) [32]byte {
	lengthRoot := merkle_tree.Uint64Root(length)
	return merkle_tree.Sha256(root[:], lengthRoot[:])
}

// chunkCount returns the chunk count for a type (used for limits)
func chunkCount(typeInfo *TypeInfo) uint64 {
	switch typeInfo.Type {
	case ssz.TypeBitVector:
		// For bitvector, chunk count is based on bits
		return uint64((typeInfo.BitLength + 255) / 256)
	case ssz.TypeBitList:
		// For bitlist, chunk count is based on max bits
		return uint64((typeInfo.BitLength + 255) / 256)
	case ssz.TypeList:
		// For lists, return max length
		return uint64(typeInfo.Length)
	case ssz.TypeVector:
		// For vectors, it depends on the element type
		if isBasicType(typeInfo.ElementType) {
			// For basic types, calculate based on packed size
			bytesPerElem := basicTypeSize(typeInfo.ElementType)
			totalBytes := typeInfo.Length * bytesPerElem
			return uint64((totalBytes + BYTES_PER_CHUNK - 1) / BYTES_PER_CHUNK)
		}
		// For composite types, each element is a chunk
		return uint64(typeInfo.Length)
	default:
		return 0
	}
}

// isBasicType returns true if the type is a basic type
func isBasicType(typeInfo *TypeInfo) bool {
	switch typeInfo.Type {
	case ssz.TypeUint8, ssz.TypeUint16, ssz.TypeUint32, ssz.TypeUint64, 
	     ssz.TypeUint128, ssz.TypeUint256, ssz.TypeBoolean:
		return true
	default:
		return false
	}
}

// basicTypeSize returns the size in bytes of a basic type
func basicTypeSize(typeInfo *TypeInfo) int {
	switch typeInfo.Type {
	case ssz.TypeUint8, ssz.TypeBoolean:
		return 1
	case ssz.TypeUint16:
		return 2
	case ssz.TypeUint32:
		return 4
	case ssz.TypeUint64:
		return 8
	case ssz.TypeUint128:
		return 16
	case ssz.TypeUint256:
		return 32
	default:
		return 0
	}
}

// hashTreeRootVector calculates the hash tree root of a vector
func hashTreeRootVector(v reflect.Value, typeInfo *TypeInfo) ([32]byte, error) {
	length := typeInfo.Length
	elemType := typeInfo.ElementType
	
	// Special case for byte arrays (Vector[uint8, N])
	if elemType.Type == ssz.TypeUint8 {
		var bytes []byte
		switch v.Kind() {
		case reflect.Array:
			bytes = make([]byte, v.Len())
			for i := 0; i < v.Len(); i++ {
				bytes[i] = uint8(v.Index(i).Uint())
			}
		case reflect.Slice:
			bytes = v.Bytes()
		default:
			return [32]byte{}, fmt.Errorf("invalid type for byte vector: %v", v.Type())
		}
		
		// For byte vectors <= 32 bytes, they're already a single chunk
		if len(bytes) <= 32 {
			var chunk [32]byte
			copy(chunk[:], bytes)
			return chunk, nil
		}
		
		// For larger byte vectors, use BytesRoot which implements pack + merkleize
		return merkle_tree.BytesRoot(bytes)
	}
	
	// For vectors of basic types (not uint8): merkleize(pack(value))
	if isBasicType(elemType) {
		chunks := packBasicVector(v, length, elemType)
		// According to the spec tests, we need to use a limit based on total byte size
		limit := uint64(length) * uint64(basicTypeSize(elemType))
		return merkle_tree.MerkleizeVector(chunks, limit)
	}
	
	// Special case for Vector[Vector[uint8, 32], N] - treat as flat byte array
	if elemType.Type == ssz.TypeVector && elemType.ElementType.Type == ssz.TypeUint8 && elemType.Length == 32 {
		// This is like BlockRoots, StateRoots - Vector of 32-byte arrays
		// Treat it as a flat byte array
		totalBytes := length * 32
		bytes := make([]byte, totalBytes)
		
		for i := 0; i < length && i < v.Len(); i++ {
			elem := v.Index(i)
			if elem.Kind() == reflect.Slice && elem.Len() == 32 {
				copy(bytes[i*32:(i+1)*32], elem.Bytes())
			}
		}
		
		return merkle_tree.BytesRoot(bytes)
	}
	
	// For vectors of composite types: merkleize([hash_tree_root(element) for element in value])
	chunks := make([][32]byte, length)
	for i := 0; i < length; i++ {
		var elem reflect.Value
		if i < v.Len() {
			elem = v.Index(i)
		} else {
			// Pad with zero values if vector is shorter
			elem = reflect.Zero(v.Type().Elem())
		}
		
		hash, err := hashTreeRoot(elem, elemType)
		if err != nil {
			return [32]byte{}, fmt.Errorf("error hashing vector element %d: %w", i, err)
		}
		chunks[i] = hash
	}
	
	return merkleize(chunks, 0)
}

// hashTreeRootList calculates the hash tree root of a list
func hashTreeRootList(v reflect.Value, typeInfo *TypeInfo) ([32]byte, error) {
	elemType := typeInfo.ElementType
	length := v.Len()
	
	// Special case for strings (list of bytes)
	if v.Kind() == reflect.String {
		bytes := []byte(v.String())
		root, err := merkle_tree.BytesRoot(bytes)
		if err != nil {
			return [32]byte{}, err
		}
		return mixInLength(root, uint64(length)), nil
	}
	
	// For lists of basic types: mix_in_length(merkleize(pack(value), limit=chunk_count(type)), len(value))
	if isBasicType(elemType) {
		var chunks [][32]byte
		
		if elemType.Type == ssz.TypeUint8 {
			// Special case for byte slices
			bytes := v.Bytes()
			chunks = packBytes(bytes)
		} else {
			// Pack other basic types
			chunks = packBasicVector(v, length, elemType)
		}
		
		// Calculate limit based on max capacity
		limit := chunkCount(typeInfo)
		if limit > 0 && elemType.Type == ssz.TypeUint8 {
			// For byte lists, limit is in chunks not elements
			limit = (uint64(typeInfo.Length) + BYTES_PER_CHUNK - 1) / BYTES_PER_CHUNK
		}
		
		root, err := merkleize(chunks, limit)
		if err != nil {
			return [32]byte{}, err
		}
		
		return mixInLength(root, uint64(length)), nil
	}
	
	// For lists of composite types: mix_in_length(merkleize([hash_tree_root(element) for element in value], limit), len(value))
	chunks := make([][32]byte, length)
	for i := 0; i < length; i++ {
		elem := v.Index(i)
		hash, err := hashTreeRoot(elem, elemType)
		if err != nil {
			return [32]byte{}, fmt.Errorf("error hashing list element %d: %w", i, err)
		}
		chunks[i] = hash
	}
	
	limit := chunkCount(typeInfo)
	root, err := merkleize(chunks, limit)
	if err != nil {
		return [32]byte{}, err
	}
	
	return mixInLength(root, uint64(length)), nil
}

// hashTreeRootContainer calculates the hash tree root of a container
func hashTreeRootContainer(v reflect.Value, typeInfo *TypeInfo) ([32]byte, error) {
	// Containers: merkleize([hash_tree_root(element) for element in value])
	// We need to convert field values to the format expected by merkle_tree.HashTreeRoot
	fieldValues := make([]any, 0, len(typeInfo.Fields))
	
	for _, field := range typeInfo.Fields {
		fieldValue := v.Field(field.Index)
		
		// For basic types, pass the value directly
		// For composite types, compute hash and pass as []byte
		if isBasicType(field.Type) {
			switch field.Type.Type {
			case ssz.TypeUint8:
				fieldValues = append(fieldValues, uint8(fieldValue.Uint()))
			case ssz.TypeUint16:
				fieldValues = append(fieldValues, uint16(fieldValue.Uint()))
			case ssz.TypeUint32:
				fieldValues = append(fieldValues, uint32(fieldValue.Uint()))
			case ssz.TypeUint64:
				fieldValues = append(fieldValues, fieldValue.Uint())
			case ssz.TypeBoolean:
				fieldValues = append(fieldValues, fieldValue.Bool())
			case ssz.TypeUint128, ssz.TypeUint256:
				// For uint256, we need to pass as bytes
				var bytes [32]byte
				if fieldValue.Type() == uint256Type {
					uint256Val := fieldValue.Interface().(uint256.Int)
					uint256Val.WriteToSlice(bytes[:])
				} else if fieldValue.Kind() == reflect.Ptr && fieldValue.Type().Elem() == uint256Type {
					if !fieldValue.IsNil() {
						uint256Val := fieldValue.Elem().Interface().(uint256.Int)
						uint256Val.WriteToSlice(bytes[:])
					}
				}
				if field.Type.Type == ssz.TypeUint128 {
					fieldValues = append(fieldValues, bytes[:16])
				} else {
					fieldValues = append(fieldValues, bytes[:])
				}
			}
		} else if field.Type.Type == ssz.TypeBitVector {
			// For bitvector, pass the bytes directly
			fieldValues = append(fieldValues, fieldValue.Bytes())
		} else {
			// For composite types, compute hash
			hash, err := hashTreeRoot(fieldValue, field.Type)
			if err != nil {
				return [32]byte{}, fmt.Errorf("error hashing field %s: %w", field.Name, err)
			}
			fieldValues = append(fieldValues, hash[:])
		}
	}
	
	return merkle_tree.HashTreeRoot(fieldValues...)
}