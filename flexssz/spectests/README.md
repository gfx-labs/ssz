# SSZ BitList and BitVector Implementation

This directory contains implementations of BitList and BitVector types for SSZ (Simple Serialize) encoding, as used in Ethereum 2.0.

## Overview

### BitList
A **BitList** is a variable-length bit array with SSZ encoding support. It uses a delimiter bit to indicate the actual length of the bit array.

Key characteristics:
- Variable length up to a specified maximum
- The last byte contains a delimiter bit (the highest set bit)
- Used in structs with `ssz:"bitlist"` tag
- Example: `AggregationBits []byte \`ssz:"bitlist" ssz-max:"2048"\``

### BitVector
A **BitVector** is a fixed-length bit array.

Key characteristics:
- Fixed length determined at creation
- No delimiter bit needed
- More space-efficient than BitList for fixed-size bit arrays

## Usage

### Creating BitLists and BitVectors

```go
// Create a BitList with 100 bits
bl := NewBitList(100)

// Create a BitVector with 256 bits  
bv := NewBitVector(256)
```

### Setting and Getting Bits

```go
// Set bit at index 10
err := bl.SetBit(10)

// Get bit at index 10
isSet, err := bl.GetBit(10)
```

### Integration with SSZ Structs

BitLists are used in Ethereum 2.0 structs like `Attestation`:

```go
type Attestation struct {
    AggregationBits []byte           `json:"aggregation_bits" ssz:"bitlist" ssz-max:"2048"`
    Data            *AttestationData `json:"data"`
    Signature       [96]byte         `json:"signature" ssz-size:"96"`
}

// Create attestation with BitList
att := &Attestation{
    AggregationBits: NewBitList(2048).Bytes(),
    // ... other fields
}
```

### Validation

```go
// Validate a BitList doesn't exceed maximum size
err := ValidateBitList(bl, 2048)
```

## BitList Format

The BitList format follows the SSZ specification:

1. **Bits are packed into bytes** - 8 bits per byte, little-endian bit order
2. **Delimiter bit** - The highest set bit in the last byte marks the end
3. **Length calculation** - `(number_of_bytes - 1) * 8 + position_of_delimiter_bit - 1`

Example:
- For a 5-bit BitList: `[0x35]` = `00110101` in binary
  - Bits 0, 2, 4 are set (reading right to left)
  - Bit 5 is the delimiter
  - Actual length = 5 bits

## Testing

Run the tests to see examples and verify functionality:

```bash
go test -v ./flexssz/spectests/
```

## Compatibility

These implementations are designed to be compatible with:
- The SSZ specification used in Ethereum 2.0
- Existing flexssz encoder/decoder functions
- The struct definitions in `struct.go` that use `ssz:"bitlist"` tags