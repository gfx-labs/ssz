# Vector Field Types in genssz

This document describes the vector field type implementation in genssz.

## Overview

Vectors are fixed-size arrays in SSZ. The genssz code generator now supports vectors of various element types beyond just `uint8` (bytevectors).

## Supported Vector Types

### Basic Numeric Vectors
- `vector[N] of uint8` (bytevector)
- `vector[N] of uint16`
- `vector[N] of uint32`
- `vector[N] of uint64`

### Reference Vectors
- `vector[N] of ref` - Vectors containing references to other SSZ types

### Nested Vectors
- `vector[N] of vector[M]` - Vectors of vectors (2D arrays)

## Schema Examples

### Basic Numeric Vectors
```yaml
structs:
  - name: NumericExample
    type: container
    children:
      - name: small_values
        type: vector
        size: 10
        children:
          - type: uint16
      - name: large_values
        type: vector
        size: 5
        children:
          - type: uint64
```

### Vector of References
```yaml
structs:
  - name: Point
    type: container
    children:
      - name: x
        type: uint32
      - name: y
        type: uint32
  
  - name: Path
    type: container
    children:
      - name: waypoints
        type: vector
        size: 10
        children:
          - type: ref
            ref: Point
```

### Nested Vectors
```yaml
structs:
  - name: Matrix
    type: container
    children:
      - name: data
        type: vector
        size: 3
        children:
          - type: vector
            size: 4
            children:
              - type: uint32
```

## Generated Code

The code generator creates efficient getter and setter methods for each vector field:

### For Numeric Vectors
```go
// Getter returns a fixed-size array
func (s TypeName) FieldName() [N]ElementType

// Setter accepts a fixed-size array
func (s TypeName) SetFieldName(v [N]ElementType)

// Index-based getter - returns element at index
func (s TypeName) FieldNameAt(idx int) ElementType

// Index-based setter - sets element at index
func (s TypeName) SetFieldNameAt(idx int, v ElementType)
```

### For Reference Vectors
```go
// Getter returns array of reference types
func (s TypeName) FieldName() [N]RefType

// Setter accepts array of reference types
func (s TypeName) SetFieldName(v [N]RefType)

// Index-based getter - returns element at index
func (s TypeName) FieldNameAt(idx int) RefType

// Index-based setter - sets element at index
func (s TypeName) SetFieldNameAt(idx int, v RefType)
```

### For Nested Vectors
```go
// Currently returns raw byte slices for each inner vector
func (s TypeName) FieldName() [N][]byte

// Setter accepts array of byte slices
func (s TypeName) SetFieldName(v [N][]byte)

// Index-based getter - returns inner vector at index
func (s TypeName) FieldNameAt(idx int) []byte  // or [M]byte for bytevectors

// Index-based setter - sets inner vector at index
func (s TypeName) SetFieldNameAt(idx int, v []byte)  // or [M]byte for bytevectors
```

### Index Access Safety
All index-based accessor methods include bounds checking and will panic if the index is out of bounds. The valid range is `[0, N)` where N is the vector size.

## Implementation Details

### Serialization
- Vectors are serialized as contiguous bytes in the container
- Elements are packed without padding between them
- Little-endian encoding for numeric types

### Optimizations
The implementation includes optimizations for vector of vectors:
1. **Aligned bytevectors**: When inner bytevectors are exactly 32 bytes, all data is copied at once
2. **Aligned basic types**: When inner vectors pack perfectly into chunks (e.g., 16-byte vectors), data is copied in bulk
3. **Single copy operation**: For these cases, we copy all vector data at once instead of iterating through each inner vector
4. This optimization leverages the fact that vectors are stored contiguously in their packed representation

### Hashing (SSZ Spec Compliant)
- Elements are packed into 32-byte chunks per SSZ specification
- For basic types (uint8, uint16, uint32, uint64):
  - Multiple elements are packed per chunk when possible
  - Example: 16 uint16 values fit in one 32-byte chunk
  - Last chunk is zero-padded if needed
- For reference types:
  - Each element is hashed individually to produce 32-byte hashes
  - These hashes form the chunks for merkleization
- Merkle tree is computed over the packed chunks
- Small vectors (≤32 bytes total) are directly used as single chunk with padding
- Nested vectors compute merkle roots recursively with proper packing

### Size Calculation
- Vector size = element_count × element_size
- Size is fixed and known at compile time
- No offset tables needed for vectors

## Limitations

1. Only fixed-size vectors are supported (no dynamic arrays)
2. Nested vectors beyond 2 levels require manual handling
3. Vector element types must be fixed-size

## Future Improvements

1. Better type safety for nested vectors
2. Support for more complex nested structures
3. Optimized hashing for specific vector sizes