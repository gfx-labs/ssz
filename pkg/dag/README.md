# DAG Package

This package provides dependency analysis and topological sorting for SSZ type definitions.

## Purpose

When generating code from SSZ schemas, types should be ordered so that:
1. Basic types (with no dependencies) appear first
2. Complex types appear after their dependencies
3. Output is deterministic (alphabetically sorted within each dependency level)

## Usage

```go
import (
    "github.com/gfx-labs/cuessz"
    "github.com/gfx-labs/ssz/pkg/dag"
)

func generateCode(defs map[string]cuessz.Def) {
    // Sort types by dependency order
    sortedNames := dag.TopologicalSort(defs)

    // Generate code in order
    for _, name := range sortedNames {
        def := defs[name]
        // ... generate code for def
    }
}
```

## Example Output

Given these types:
```yaml
defs:
  Root:        # vector of uint8
  Signature:   # vector of uint8
  Checkpoint:  # container with Root field
  AttestationData: # container with Checkpoint field
  Attestation: # container with AttestationData field
```

The output order will be:
```
1. Root          (Level 0: no dependencies)
2. Signature     (Level 0: no dependencies)
3. Checkpoint    (Level 1: depends on Root)
4. AttestationData (Level 2: depends on Checkpoint)
5. Attestation   (Level 3: depends on AttestationData)
```

Within each level, types are sorted alphabetically for determinism.

## Functions

### `TopologicalSort(defs map[string]cuessz.Def) []string`

Returns type names in dependency order (least dependent first).

### `CollectDependencies(def cuessz.Def, defs map[string]cuessz.Def) []string`

Returns all type names that a definition depends on (transitively).

## Cycle Detection

If circular dependencies are detected (e.g., A → B → A), types involved in the cycle are assigned the same dependency level and sorted alphabetically.
