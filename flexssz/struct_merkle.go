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

// HashTreeRoot calculates the merkle root of a value based on its type and struct tags
func HashTreeRoot(v any) ([32]byte, error) {
	rv := reflect.ValueOf(v)

	// Handle pointer by dereferencing
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return [32]byte{}, fmt.Errorf("cannot hash nil pointer")
		}
		rv = rv.Elem()
	}

	// Get type info
	typeInfo, err := GetTypeInfo(rv.Type(), nil)
	if err != nil {
		return [32]byte{}, fmt.Errorf("error getting type info: %w", err)
	}

	// Calculate hash tree root for any type
	return hashTreeRoot(rv, typeInfo)
}

// hashTreeRoot implements the recursive hash_tree_root function from the SSZ spec
func hashTreeRoot(v reflect.Value, typeInfo *TypeInfo) (out [32]byte, err error) {
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
		// Basic types: directly compute hash of the value
		return hashTreeRootBasicValue(v, typeInfo)

	case ssz.TypeBitVector:
		// Bitvectors: merkleize(pack_bits(value), limit=chunk_count(type))
		if v.Kind() != reflect.Slice || v.Type().Elem().Kind() != reflect.Uint8 {
			return [32]byte{}, fmt.Errorf("invalid type for bitvector: %v", v.Type())
		}
		chunks := packBytes(v.Bytes())
		err := merkle_tree.MerklizeChunks(chunks, out[:])
		if err != nil {
			return [32]byte{}, err
		}
		return out, nil

	case ssz.TypeBitList:
		// Bitlists: mix_in_length(merkleize(pack_bits(value), limit=chunk_count(type)), len(value))
		if v.Kind() != reflect.Slice || v.Type().Elem().Kind() != reflect.Uint8 {
			return [32]byte{}, fmt.Errorf("invalid type for bitlist: %v", v.Type())
		}
		return merkle_tree.BitlistRootWithLimit(v.Bytes(), uint64(typeInfo.BitLength))

	case ssz.TypeVector:
		return hashTreeRootVector(v, typeInfo)

	case ssz.TypeList:
		return hashTreeRootList(v, typeInfo)

	case ssz.TypeContainer:
		return hashTreeRootContainer(v, typeInfo)

	default:
		return [32]byte{}, fmt.Errorf("unsupported SSZ type for merkle root: %v", typeInfo.Type)
	}
}

// hashTreeRootBasicValue computes the hash tree root of a single basic value
func hashTreeRootBasicValue(v reflect.Value, typeInfo *TypeInfo) ([32]byte, error) {
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

	// For basic values, the hash is just the chunk itself (no merkleization needed)
	return chunk, nil
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

		err := merkle_tree.MerklizeChunks(chunks, chunks[0][:])
		if err != nil {
			return [32]byte{}, err
		}

		return chunks[0], nil
	}

	// Special case for Vector[Vector[uint8, 32], N] - each 32-byte vector is already a chunk
	if elemType.Type == ssz.TypeVector && elemType.ElementType.Type == ssz.TypeUint8 && elemType.Length == 32 {
		// Each 32-byte array is already a chunk
		chunks := make([][32]byte, length)

		for i := 0; i < length && i < v.Len(); i++ {
			elem := v.Index(i)
			if elem.Kind() == reflect.Slice && elem.Len() == 32 {
				copy(chunks[i][:], elem.Bytes())
			}
		}

		err := merkle_tree.MerklizeChunks(chunks, chunks[0][:])
		if err != nil {
			return [32]byte{}, err
		}
		return chunks[0], nil
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

	err := merkle_tree.MerklizeChunks(chunks, chunks[0][:])
	if err != nil {
		return [32]byte{}, err
	}
	return chunks[0], nil
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

	if length == 0 {
		if isBasicType(elemType) {
			size := (typeInfo.Length*elemType.FixedSize + 31) / 32
			return mixInLength(merkle_tree.ZeroHash(merkle_tree.GetDepth(uint64(size))), uint64(length)), nil
		}
		return mixInLength(merkle_tree.ZeroHash(merkle_tree.GetDepth(uint64(typeInfo.Length))), uint64(length)), nil
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

		err := merkle_tree.MerklizeChunks(chunks, chunks[0][:])
		if err != nil {
			return [32]byte{}, err
		}

		return mixInLength(chunks[0], uint64(length)), nil
	}

	// For lists of composite types: mix_in_length(merkleize([hash_tree_root(element) for element in value], limit), len(value))
	chunks := make([][32]byte, length)
	for i := range length {
		elem := v.Index(i)
		hash, err := hashTreeRoot(elem, elemType)
		if err != nil {
			return [32]byte{}, fmt.Errorf("error hashing list element %d: %w", i, err)
		}
		chunks[i] = hash
	}

	err := merkle_tree.MerklizeChunks(chunks, chunks[0][:])
	if err != nil {
		return [32]byte{}, err
	}

	return mixInLength(chunks[0], uint64(length)), nil
}

// hashTreeRootContainer calculates the hash tree root of a container
func hashTreeRootContainer(v reflect.Value, typeInfo *TypeInfo) ([32]byte, error) {
	// Containers: merkleize([hash_tree_root(element) for element in value])
	chunks := make([][32]byte, len(typeInfo.Fields))

	for i, field := range typeInfo.Fields {
		fieldValue := v.Field(field.Index)
		var err error
		chunks[i], err = hashTreeRoot(fieldValue, field.Type)
		if err != nil {
			return [32]byte{}, fmt.Errorf("error hashing field %s: %w", field.Name, err)
		}
	}
	err := merkle_tree.MerklizeChunks(chunks, chunks[0][:])
	if err != nil {
		return [32]byte{}, err
	}
	return chunks[0], nil
}
