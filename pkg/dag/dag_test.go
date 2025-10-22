package dag

import (
	"testing"

	"github.com/gfx-labs/cuessz"
)

func TestTopologicalSort(t *testing.T) {
	tests := []struct {
		name     string
		defs     map[string]cuessz.Def
		expected []string
	}{
		{
			name: "simple linear dependency",
			defs: map[string]cuessz.Def{
				"C": {
					Type: cuessz.TypeContainer,
					Children: []cuessz.Field{
						{
							Name: "b",
							Def: cuessz.Def{
								Type: cuessz.TypeRef,
								Ref:  "B",
							},
						},
					},
				},
				"B": {
					Type: cuessz.TypeContainer,
					Children: []cuessz.Field{
						{
							Name: "a",
							Def: cuessz.Def{
								Type: cuessz.TypeRef,
								Ref:  "A",
							},
						},
					},
				},
				"A": {
					Type: cuessz.TypeVector,
					Size: 32,
					Children: []cuessz.Field{
						{
							Name: "element",
							Def: cuessz.Def{
								Type: cuessz.TypeUint8,
							},
						},
					},
				},
			},
			expected: []string{"A", "B", "C"},
		},
		{
			name: "multiple at same level",
			defs: map[string]cuessz.Def{
				"Root": {
					Type: cuessz.TypeVector,
					Size: 32,
					Children: []cuessz.Field{
						{
							Name: "element",
							Def: cuessz.Def{
								Type: cuessz.TypeUint8,
							},
						},
					},
				},
				"Signature": {
					Type: cuessz.TypeVector,
					Size: 96,
					Children: []cuessz.Field{
						{
							Name: "element",
							Def: cuessz.Def{
								Type: cuessz.TypeUint8,
							},
						},
					},
				},
				"Container": {
					Type: cuessz.TypeContainer,
					Children: []cuessz.Field{
						{
							Name: "root",
							Def: cuessz.Def{
								Type: cuessz.TypeRef,
								Ref:  "Root",
							},
						},
						{
							Name: "sig",
							Def: cuessz.Def{
								Type: cuessz.TypeRef,
								Ref:  "Signature",
							},
						},
					},
				},
			},
			// Root and Signature are at level 0 (alphabetically sorted)
			// Container is at level 1
			expected: []string{"Root", "Signature", "Container"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TopologicalSort(tt.defs)
			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d types, got %d", len(tt.expected), len(result))
			}
			for i, name := range result {
				if name != tt.expected[i] {
					t.Errorf("position %d: expected %s, got %s", i, tt.expected[i], name)
				}
			}
		})
	}
}

func TestCollectDependencies(t *testing.T) {
	defs := map[string]cuessz.Def{
		"Root": {
			Type: cuessz.TypeVector,
			Size: 32,
		},
		"Signature": {
			Type: cuessz.TypeVector,
			Size: 96,
		},
	}

	tests := []struct {
		name     string
		def      cuessz.Def
		expected []string
	}{
		{
			name: "no dependencies",
			def: cuessz.Def{
				Type: cuessz.TypeUint64,
			},
			expected: []string{},
		},
		{
			name: "single ref",
			def: cuessz.Def{
				Type: cuessz.TypeContainer,
				Children: []cuessz.Field{
					{
						Name: "root",
						Def: cuessz.Def{
							Type: cuessz.TypeRef,
							Ref:  "Root",
						},
					},
				},
			},
			expected: []string{"Root"},
		},
		{
			name: "multiple refs",
			def: cuessz.Def{
				Type: cuessz.TypeContainer,
				Children: []cuessz.Field{
					{
						Name: "root",
						Def: cuessz.Def{
							Type: cuessz.TypeRef,
							Ref:  "Root",
						},
					},
					{
						Name: "sig",
						Def: cuessz.Def{
							Type: cuessz.TypeRef,
							Ref:  "Signature",
						},
					},
				},
			},
			expected: []string{"Root", "Signature"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CollectDependencies(tt.def, defs)
			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d dependencies, got %d", len(tt.expected), len(result))
			}
			// Convert to map for easier comparison (order doesn't matter)
			resultMap := make(map[string]bool)
			for _, dep := range result {
				resultMap[dep] = true
			}
			for _, exp := range tt.expected {
				if !resultMap[exp] {
					t.Errorf("expected dependency %s not found", exp)
				}
			}
		})
	}
}
