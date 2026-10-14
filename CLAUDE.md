# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

This repository provides SSZ (Simple Serialize) code generation tools. It consists of:

1. **genssz** - CLI tool for generating code from SSZ schema definitions
2. **gostruct** - Generator for standard Go structs with SSZ tags (compatible with fastssz, dynamic-ssz)
3. **solidssz** - Generator for immutable, fixed-size SSZ types stored as contiguous byte slices (inspired by erigon/caplin)
4. **pyssz** - Generator for Python classes compatible with [py-ssz](https://github.com/ethereum/py-ssz)
5. Supporting libraries for merkle tree operations and dependency analysis

The project uses [cuessz](https://github.com/gfx-labs/cuessz) for schema validation and definition.

## Build and Development Commands

### Build the Generator

```bash
# Build genssz binary
make build
# or
go build -o genssz ./cmd/genssz
```

### Generate Examples

```bash
# Generate all examples (consensus, zoo, progressive)
# Each example generates gostruct, solidssz, and pyssz outputs
make examples

# Generate specific examples
make consensus   # Ethereum Beacon Chain types (all generators)
make zoo         # Zoo example (all generators)
make progressive # EIP-7495 progressive containers (all generators)

# Clean generated files
make clean
```

### Run Tests

```bash
# Run all tests
go test ./...

# Run tests for specific package
go test ./pkg/merkletree/...
go test ./pkg/dag/...

# Run specific test
go test -run TestTopologicalSort ./pkg/dag/...

# Run tests with verbose output
go test -v ./...
```

### Manual Code Generation

```bash
# Generate gostruct (standard Go structs with SSZ tags)
./genssz gostruct -o output.go schema.yaml

# Generate solidssz (optimized fixed-size types)
./genssz solidssz -o output_dir/ schema.yaml

# Generate pyssz (Python classes for py-ssz)
./genssz pyssz -o output.py schema.yaml

# Multiple input schemas
./genssz gostruct -o output.go schema1.yaml schema2.json
```

## Architecture

### Generator Pipeline

All generators (`gostruct`, `solidssz`, and `pyssz`) follow this flow:

1. **Schema Loading** (`ReadSchemaFromBytes`)
   - Accepts YAML or JSON input
   - Converts YAML → JSON → cuessz validation
   - Schema validated against `reference/ssz_schema.cue`

2. **Dependency Resolution** (`pkg/dag`)
   - Topologically sorts types by dependencies
   - Ensures types are generated in correct order
   - Handles circular dependencies

3. **Code Generation**
   - `gostruct`: Generates standard Go structs with SSZ annotations
   - `solidssz`: Generates byte-oriented accessors for fixed-size types
   - `pyssz`: Generates Python classes for py-ssz library

### Package Structure

- **`cmd/genssz`** - CLI entry point using Kong command parser
- **`pkg/gen/generators/gostructgen`** - Standard Go struct generator
  - Produces structs compatible with fastssz, dynamic-ssz
  - Minimal SSZ tags (only where necessary)
  - Uses `github.com/dave/jennifer` for code generation
- **`pkg/gen/generators/solidsszgen`** - Immutable byte-slice generator
  - Only supports fixed-size types
  - Memory-efficient for large data structures
  - Based on erigon/caplin strategy
- **`pkg/gen/generators/pysszgen`** - Python py-ssz generator
  - Generates classes for `HashableContainer` and `SignedHashableContainer`
  - Imports standard bytes types (bytes4, bytes20, bytes32, bytes48, bytes96)
  - Compatible with py-ssz library
- **`pkg/dag`** - Dependency analysis and topological sorting
  - Ensures types are ordered correctly
  - Alphabetically sorts within dependency levels
- **`pkg/merkletree`** - Merkle tree operations
  - Based on erigon's merkle_tree implementation
  - Includes buffer pooling for performance

### Schema Format

Schemas are validated by cuessz using the CUE schema at `reference/ssz_schema.cue`. Key schema structure:

```yaml
version: 1.0.0
metadata:
  namespace: examples.myspec  # Becomes package name
  description: Optional description

defs:
  TypeName:
    type: container|vector|list|bitvector|bitlist|uint8|uint16|uint32|uint64|uint128|uint256|boolean|ref|progressive_container
    children: [...]  # For containers, vectors, lists
    size: N          # For vectors, bitvectors
    limit: N         # For lists, bitlists (max 2^32)
```

### Code Generation Details

**gostruct output:**
- Fixed-size arrays for vectors/bitvectors: `[N]Type`
- Pointers for container references: `*ContainerType`
- SSZ tags only where required (variable-length types need `ssz-max`)
- Type aliases for reusable types

**solidssz output:**
- Only supports fixed-size types (no lists, bitlists, variable containers)
- Generates one file per type
- Stores data as `[]byte` internally
- Provides getter/setter methods
- Memory-efficient for read-heavy workloads

**pyssz output:**
- Classes inherit from `HashableContainer` or `SignedHashableContainer`
- `SignedHashableContainer` used when container has a `signature` field
- Standard bytes types (bytes4, bytes20, bytes32, bytes48, bytes96) imported from `ssz.sedes`
- Fields defined as class-level `fields` list with tuples of (name, type)
- Supports all SSZ types: List, Vector, Bitlist, Bitvector, basic types

### Dependencies

- **cuessz** - Schema validation and parsing (CUE-based)
- **jennifer** - Go code generation library
- **gohashtree** - Fast hashing for merkle trees
- **dynamic-ssz** - Runtime SSZ operations
- **erigon** - Reference for solidssz implementation

## Key Concepts

### SSZ Type System

SSZ supports:
- **Basic types**: uint8, uint16, uint32, uint64, uint128, uint256, boolean
- **Composite types**: container, vector, list, union, bitvector, bitlist
- **Progressive containers** (EIP-7495): Stable containers with optional fields

### Generator Selection

Use **gostruct** when:
- Need compatibility with fastssz or dynamic-ssz
- Types include variable-length fields (lists, bitlists)
- Standard Go struct representation is desired

Use **solidssz** when:
- All types are fixed-size
- Memory efficiency is critical
- Data is mostly read-only
- Working with large data structures (e.g., Beacon State)

Use **pyssz** when:
- Need Python implementation using py-ssz library
- Building Python tools for Ethereum consensus specs
- Interoperating with Python SSZ libraries
- All SSZ types are supported (containers, lists, vectors, etc.)

### Schema Validation

All schemas are validated at generation time using cuessz. Common validation:
- Type constraints (size/limit within uint32 range)
- Progressive container active_fields correctness
- Children structure matches type requirements

## Examples

Example schemas are in `examples/specs/`:
- **consensus** - Full Ethereum Beacon Chain types
- **zoo** - Simple example from py-ssz with nested containers
- **progressive** - EIP-7495 progressive/stable containers

Generated output in `examples/output/`:
- `examples/output/{example}/gostruct/generated.go` - Go structs
- `examples/output/{example}/solidssz/*_generated.go` - Solidssz types
- `examples/output/{example}/pyssz/generated.py` - Python classes

## Testing Strategy

Test files are co-located with source:
- `pkg/merkletree/*_test.go` - Merkle tree operations
- `pkg/dag/dag_test.go` - Dependency resolution

The old test infrastructure (`flexssz/`, `genssz/test_genssz/`) has been removed in favor of the new example-based approach.
