package dag

import (
	"sort"

	"github.com/gfx-labs/cuessz"
)

// TopologicalSort sorts type definitions by dependency order.
// Types with no dependencies come first, then types that depend on them, etc.
// Within each dependency level, types are sorted alphabetically for determinism.
//
// This ensures that:
// - Basic types (Root, Signature, etc.) appear first
// - Complex types appear after their dependencies
// - Output is deterministic (same input = same output)
func TopologicalSort(defs map[string]cuessz.Def) []string {
	// Build dependency graph: defName -> list of defs it depends on
	deps := make(map[string][]string)
	for defName, def := range defs {
		deps[defName] = CollectDependencies(def, defs)
	}

	// Calculate dependency level for each type (0 = no deps, 1 = depends on level 0, etc.)
	levels := make(map[string]int)
	for defName := range defs {
		calculateLevel(defName, deps, levels, make(map[string]bool))
	}

	// Group by level
	levelGroups := make(map[int][]string)
	for defName, level := range levels {
		levelGroups[level] = append(levelGroups[level], defName)
	}

	// Sort within each level alphabetically, then combine
	var result []string
	maxLevel := 0
	for level := range levelGroups {
		if level > maxLevel {
			maxLevel = level
		}
	}

	for level := 0; level <= maxLevel; level++ {
		group := levelGroups[level]
		sort.Strings(group)
		result = append(result, group...)
	}

	return result
}

// CollectDependencies returns all type names that this def depends on.
// This traverses the def structure and collects all refs.
func CollectDependencies(def cuessz.Def, defs map[string]cuessz.Def) []string {
	var deps []string
	seen := make(map[string]bool)

	var collect func(d cuessz.Def)
	collect = func(d cuessz.Def) {
		switch d.Type {
		case cuessz.TypeRef:
			if d.Ref != "" && !seen[d.Ref] {
				seen[d.Ref] = true
				deps = append(deps, d.Ref)
			}
		case cuessz.TypeContainer, cuessz.TypeProgressiveContainer, cuessz.TypeVector, cuessz.TypeList:
			// Check children for refs
			for _, child := range d.Children {
				collect(child.Def)
			}
		}
	}

	collect(def)
	return deps
}

// calculateLevel computes the dependency level for a type.
// Level 0 = no dependencies
// Level N = depends on types at level N-1 or lower
func calculateLevel(defName string, deps map[string][]string, levels map[string]int, visiting map[string]bool) int {
	// Already calculated
	if level, ok := levels[defName]; ok {
		return level
	}

	// Cycle detection
	if visiting[defName] {
		// Circular dependency - assign same level
		return 0
	}

	visiting[defName] = true
	defer delete(visiting, defName)

	// Calculate level based on dependencies
	maxDepLevel := -1
	for _, dep := range deps[defName] {
		depLevel := calculateLevel(dep, deps, levels, visiting)
		if depLevel > maxDepLevel {
			maxDepLevel = depLevel
		}
	}

	level := maxDepLevel + 1
	levels[defName] = level
	return level
}
