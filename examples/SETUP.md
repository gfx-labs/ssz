# Examples Module Setup

This directory is a separate Go module that demonstrates the use of the SSZ code generators.

## Module Structure

The `examples` directory is set up as its own Go module with a **replace directive** that always uses the local ssz package from the parent directory:

```go
module github.com/gfx-labs/ssz/examples

replace github.com/gfx-labs/ssz => ../

require github.com/gfx-labs/ssz v0.0.0-00010101000000-000000000000
```

This means:
- The examples module can import packages from `github.com/gfx-labs/ssz/*`
- All imports will use the local version in `../` (the parent directory)
- Changes to the parent ssz package are immediately available to examples
- No need to publish or version the ssz package for local development

## Running Examples

From the repository root:

```bash
# Generate examples
make examples

# Or generate specific examples
make zoo
make consensus
make progressive
```

## Testing

The examples module includes tests that verify the local replace directive works correctly:

```bash
cd examples
go test -v .
```

## Generated Code

Generated code is placed in:
- `output/*/gostruct/` - Standard Go structs compatible with fastssz/dynamic-ssz
- `output/*/solidssz/` - Optimized byte-slice based SSZ types

The generated code imports `github.com/gfx-labs/ssz/pkg/solidssz` which resolves to the local package via the replace directive.
