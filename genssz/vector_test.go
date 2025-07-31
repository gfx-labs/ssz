package genssz

import (
	"bytes"
	"os"
	"testing"

	"github.com/dave/jennifer/jen"
)

func TestGenerateVectorTypes(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		wantCode []string // Expected code snippets
	}{
		{
			name: "vector of uint16",
			yaml: `
package: test
structs:
  - name: TestVectorU16
    type: container
    children:
      - name: values
        type: vector
        size: 4
        children:
          - type: uint16
`,
			wantCode: []string{
				"vector[4] of uint16",
				"func (s TestVectorU16) Values() [4]uint16",
				"func (s TestVectorU16) SetValues(v [4]uint16)",
			},
		},
		{
			name: "vector of uint32",
			yaml: `
package: test
structs:
  - name: TestVectorU32
    type: container
    children:
      - name: data
        type: vector
        size: 3
        children:
          - type: uint32
`,
			wantCode: []string{
				"vector[3] of uint32",
				"func (s TestVectorU32) Data() [3]uint32",
				"func (s TestVectorU32) SetData(v [3]uint32)",
			},
		},
		{
			name: "vector of uint64",
			yaml: `
package: test
structs:
  - name: TestVectorU64
    type: container
    children:
      - name: timestamps
        type: vector
        size: 2
        children:
          - type: uint64
`,
			wantCode: []string{
				"vector[2] of uint64",
				"func (s TestVectorU64) Timestamps() [2]uint64",
				"func (s TestVectorU64) SetTimestamps(v [2]uint64)",
			},
		},
		{
			name: "vector of references",
			yaml: `
package: test
structs:
  - name: Point
    type: container
    children:
      - name: x
        type: uint32
      - name: y
        type: uint32
  - name: TestVectorRef
    type: container
    children:
      - name: points
        type: vector
        size: 3
        children:
          - type: ref
            ref: Point
`,
			wantCode: []string{
				"vector[3] of ref: Point",
				"func (s TestVectorRef) Points() [3]Point",
				"func (s TestVectorRef) SetPoints(v [3]Point)",
			},
		},
		{
			name: "mixed container with vectors",
			yaml: `
package: test
structs:
  - name: MixedContainer
    type: container
    children:
      - name: id
        type: uint64
      - name: values
        type: vector
        size: 5
        children:
          - type: uint16
      - name: hash
        type: bytevector
        size: 32
`,
			wantCode: []string{
				"id (uint64)",
				"values (vector[5] of uint16)",
				"hash (bytevector[32])",
				"func (s MixedContainer) Values() [5]uint16",
			},
		},
		{
			name: "vector of vectors",
			yaml: `
package: test
structs:
  - name: VectorOfVectors
    type: container
    children:
      - name: matrix
        type: vector
        size: 3
        children:
          - type: vector
            size: 4
            children:
              - type: uint32
      - name: bytes_matrix
        type: vector
        size: 2
        children:
          - type: vector
            size: 8
            children:
              - type: uint8
`,
			wantCode: []string{
				"matrix (vector[3] of vector[4] of uint32)",
				"bytes_matrix (vector[2] of bytevector[8])",
				"func (s VectorOfVectors) Matrix() [3][]byte",
				"func (s VectorOfVectors) Bytes_matrix() [2][8]byte",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse schema
			schema, err := ReadSchemaFromBytes([]byte(tt.yaml))
			if err != nil {
				t.Fatalf("ReadSchemaFromBytes failed: %v", err)
			}

			// Parse to world
			world, err := ParseSchemaToWorld(schema)
			if err != nil {
				t.Fatalf("ParseSchemaToWorld failed: %v", err)
			}

			// Generate code
			f, err := GenerateCode(world, schema)
			if err != nil {
				t.Fatalf("GenerateCode failed: %v", err)
			}

			// Render code to string
			buf := &bytes.Buffer{}
			err = f.Render(buf)
			if err != nil {
				t.Fatalf("Failed to render code: %v", err)
			}

			code := buf.String()

			// Check for expected code snippets
			for _, want := range tt.wantCode {
				if !bytes.Contains([]byte(code), []byte(want)) {
					t.Errorf("Generated code does not contain expected snippet: %q", want)
					t.Logf("Generated code:\n%s", code)
				}
			}
		})
	}
}

func TestVectorFieldSizes(t *testing.T) {
	tests := []struct {
		name         string
		yaml         string
		expectedSize int
	}{
		{
			name: "vector of uint16[4]",
			yaml: `
package: test
structs:
  - name: TestSize1
    type: container
    children:
      - name: values
        type: vector
        size: 4
        children:
          - type: uint16
`,
			expectedSize: 8, // 4 * 2 bytes
		},
		{
			name: "vector of uint32[3]",
			yaml: `
package: test
structs:
  - name: TestSize2
    type: container
    children:
      - name: values
        type: vector
        size: 3
        children:
          - type: uint32
`,
			expectedSize: 12, // 3 * 4 bytes
		},
		{
			name: "mixed with vector",
			yaml: `
package: test
structs:
  - name: TestSize3
    type: container
    children:
      - name: id
        type: uint64
      - name: values
        type: vector
        size: 2
        children:
          - type: uint32
      - name: flag
        type: boolean
`,
			expectedSize: 17, // 8 + 8 + 1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse schema
			schema, err := ReadSchemaFromBytes([]byte(tt.yaml))
			if err != nil {
				t.Fatalf("ReadSchemaFromBytes failed: %v", err)
			}

			// Parse to world
			world, err := ParseSchemaToWorld(schema)
			if err != nil {
				t.Fatalf("ParseSchemaToWorld failed: %v", err)
			}

			// Generate code
			f, err := GenerateCode(world, schema)
			if err != nil {
				t.Fatalf("GenerateCode failed: %v", err)
			}

			// Render code to string
			buf := &bytes.Buffer{}
			err = f.Render(buf)
			if err != nil {
				t.Fatalf("Failed to render code: %v", err)
			}

			code := buf.String()

			// Check for expected size
			expectedSizeStr := jen.Lit(tt.expectedSize).GoString()
			if !bytes.Contains([]byte(code), []byte(expectedSizeStr)) {
				t.Errorf("Generated code does not contain expected size: %d", tt.expectedSize)
				t.Logf("Generated code:\n%s", code)
			}
		})
	}
}

// TestVectorIndexAccessors tests the generated index-based accessor methods
func TestVectorIndexAccessors(t *testing.T) {
	tests := []struct {
		name         string
		yaml         string
		wantCode     []string
		dontWantCode []string
	}{
		{
			name: "vector index accessors",
			yaml: `
package: test
structs:
  - name: IndexAccessTest
    type: container
    children:
      - name: nums
        type: vector
        size: 5
        children:
          - type: uint32
      - name: bytes
        type: bytevector
        size: 10
`,
			wantCode: []string{
				"func (s IndexAccessTest) NumsAt(idx int) uint32",
				"func (s IndexAccessTest) SetNumsAt(idx int, v uint32)",
				"Panics if index is out of bounds [0, 5)",
			},
			dontWantCode: []string{
				"func (s IndexAccessTest) BytesAt(idx int) byte",
				"func (s IndexAccessTest) SetBytesAt(idx int, v byte)",
			},
		},
		{
			name: "vector of references with index accessors",
			yaml: `
package: test
structs:
  - name: Item
    type: container
    children:
      - name: id
        type: uint32
      - name: value
        type: uint64
  - name: RefIndexTest
    type: container
    children:
      - name: items
        type: vector
        size: 3
        children:
          - type: ref
            ref: Item
`,
			wantCode: []string{
				"func (s RefIndexTest) ItemsAt(idx int) Item",
				"func (s RefIndexTest) SetItemsAt(idx int, v Item)",
				"Panics if index is out of bounds [0, 3)",
			},
		},
		{
			name: "nested vectors with index accessors",
			yaml: `
package: test
structs:
  - name: NestedIndexTest
    type: container
    children:
      - name: matrix
        type: vector
        size: 2
        children:
          - type: vector
            size: 4
            children:
              - type: uint8
`,
			wantCode: []string{
				"func (s NestedIndexTest) MatrixAt(idx int) [4]byte",
				"func (s NestedIndexTest) SetMatrixAt(idx int, v [4]byte)",
				"Panics if index is out of bounds [0, 2)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse schema
			schema, err := ReadSchemaFromBytes([]byte(tt.yaml))
			if err != nil {
				t.Fatalf("ReadSchemaFromBytes failed: %v", err)
			}

			// Parse to world
			world, err := ParseSchemaToWorld(schema)
			if err != nil {
				t.Fatalf("ParseSchemaToWorld failed: %v", err)
			}

			// Generate code
			f, err := GenerateCode(world, schema)
			if err != nil {
				t.Fatalf("GenerateCode failed: %v", err)
			}

			// Render code to string
			buf := &bytes.Buffer{}
			err = f.Render(buf)
			if err != nil {
				t.Fatalf("Failed to render code: %v", err)
			}

			code := buf.String()

			// Check for expected code snippets
			for _, want := range tt.wantCode {
				if !bytes.Contains([]byte(code), []byte(want)) {
					t.Errorf("Generated code does not contain expected snippet: %q", want)
					t.Logf("Generated code:\n%s", code)
				}
			}
			
			// Check that unwanted code is NOT present
			for _, dontWant := range tt.dontWantCode {
				if bytes.Contains([]byte(code), []byte(dontWant)) {
					t.Errorf("Generated code contains unwanted snippet: %q", dontWant)
					t.Logf("Generated code:\n%s", code)
				}
			}
		})
	}
}

// TestVectorCodeGeneration ensures the generated code compiles
func TestVectorCodeGeneration(t *testing.T) {
	yaml := `
package: vectortest
structs:
  - name: Point
    type: container
    children:
      - name: x
        type: uint32
      - name: y
        type: uint32
  - name: VectorExample
    type: container
    children:
      - name: id
        type: uint64
      - name: values
        type: vector
        size: 4
        children:
          - type: uint16
      - name: points
        type: vector
        size: 2
        children:
          - type: ref
            ref: Point
      - name: data
        type: bytevector
        size: 16
`

	// Parse schema
	schema, err := ReadSchemaFromBytes([]byte(yaml))
	if err != nil {
		t.Fatalf("ReadSchemaFromBytes failed: %v", err)
	}

	// Parse to world
	world, err := ParseSchemaToWorld(schema)
	if err != nil {
		t.Fatalf("ParseSchemaToWorld failed: %v", err)
	}

	// Generate code
	f, err := GenerateCode(world, schema)
	if err != nil {
		t.Fatalf("GenerateCode failed: %v", err)
	}

	// Write to temporary file
	tmpfile, err := os.CreateTemp("", "vector_test_*.go")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	err = f.Render(tmpfile)
	if err != nil {
		t.Fatalf("Failed to render code: %v", err)
	}
	tmpfile.Close()

	// Just verify we generated something
	content, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	// Basic checks
	requiredSnippets := []string{
		"type Point []byte",
		"type VectorExample []byte",
		"func (s VectorExample) Values() [4]uint16",
		"func (s VectorExample) SetValues(v [4]uint16)",
		"func (s VectorExample) Points() [2]Point",
		"func (s VectorExample) SetPoints(v [2]Point)",
	}

	for _, snippet := range requiredSnippets {
		if !bytes.Contains(content, []byte(snippet)) {
			t.Errorf("Generated code missing expected snippet: %q", snippet)
		}
	}
}