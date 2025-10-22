# SSZ Examples

This directory contains example SSZ schema definitions and generated code from the [cuessz](https://github.com/gfx-labs/cuessz) repository.

## Directory Structure

```
examples/
├── specs/              # Schema definitions (YAML format)
│   ├── consensus/     # Ethereum Beacon Chain types
│   │   └── schema.yaml
│   ├── zoo/           # Zoo example (from py-ssz)
│   │   └── schema.yaml
│   └── progressive/   # Progressive container examples (EIP-7495)
│       └── schema.yaml
│
└── output/            # Generated code
    ├── consensus/     # Generated from consensus spec
    │   └── gostruct/
    │       └── generated.go
    ├── zoo/           # Generated from zoo spec
    │   └── gostruct/
    │       └── generated.go
    └── progressive/   # Generated from progressive spec
        └── gostruct/
            └── generated.go
```

## Examples

### Consensus
Full Ethereum Beacon Chain SSZ types including:
- Basic types (Root, Signature, BLSPubkey)
- Core types (Validator, Checkpoint, BeaconBlock, etc.)
- All fork variants (Phase0, Altair, Bellatrix, Capella, Deneb)
- Light client types
- ExecutionPayload types

### Zoo
Fun example from [py-ssz](https://github.com/ethereum/py-ssz/blob/v0.5.2/ssz/examples/zoo.py) demonstrating:
- Type aliases (bytes32, bytes48, bytes96)
- Nested containers (Animal contains ClockInRecords)
- Lists of containers
- Bitvectors and bitlists

### Progressive
Examples of EIP-7495 progressive/stable containers:
- SimpleMessage → ExtendedMessage (adding fields)
- UserProfile → ExtendedUserProfile (adding fields)
- Transaction → ExtendedTransaction (adding fields)
- Config → ExtendedConfig (adding fields)

## Usage

Generate Go structs from any schema:

```bash
# From repo root
./genssz gostruct -o examples/output/zoo/gostruct/generated.go examples/specs/zoo/schema.yaml
```

The generator accepts both YAML and JSON formats. All examples use YAML for readability.

## Package Names

The generator automatically sanitizes namespace strings to valid Go package names:
- `"eth2.beacon"` → `package beacon`
- `"examples.zoo"` → `package zoo`
- `"examples.progressive"` → `package progressive`

The last component after the final dot is used as the package name.

## Generated Code Features

The `gostruct` generator produces clean Go code:

✅ **Minimal tags** - Size is implicit in fixed-size types
```go
Signature [96]uint8 `json:"signature"`        // No ssz-size tag needed
```

✅ **Only necessary SSZ annotations**
```go
ClockInRecords []*ClockInRecords `json:"clock_in_records" ssz-max:"4294967296"`  // Variable-length needs max
```

✅ **Type aliases** - Creates reusable types
```go
type bytes32 [32]uint8
type Root bytes32
```

✅ **Proper pointers** - Container references use pointers
```go
type SignedBeaconBlock struct {
    Message   *BeaconBlock `json:"message"`
    Signature Signature    `json:"signature"`
}
```

## Schema Format

Schemas use the cuessz format validated against `ssz_schema.cue` at the repo root. Example:

```yaml
version: 1.0.0
metadata:
  namespace: examples.myspec
  description: My custom types

defs:
  MyContainer:
    type: container
    children:
      - name: my_field
        def:
          type: uint64
```

All schemas are validated by cuessz during code generation, providing context-aware error messages.
