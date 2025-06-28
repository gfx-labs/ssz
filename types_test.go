package ssz

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsValid_BasicTypes(t *testing.T) {
	tests := []struct {
		name    string
		field   Field
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid uint8",
			field: Field{
				Name: "myUint8",
				Type: TypeUint8,
			},
			wantErr: false,
		},
		{
			name: "valid uint16",
			field: Field{
				Name: "myUint16",
				Type: TypeUint16,
			},
			wantErr: false,
		},
		{
			name: "valid uint32",
			field: Field{
				Name: "myUint32",
				Type: TypeUint32,
			},
			wantErr: false,
		},
		{
			name: "valid uint64",
			field: Field{
				Name: "myUint64",
				Type: TypeUint64,
			},
			wantErr: false,
		},
		{
			name: "valid uint128",
			field: Field{
				Name: "myUint128",
				Type: TypeUint128,
			},
			wantErr: false,
		},
		{
			name: "valid uint256",
			field: Field{
				Name: "myUint256",
				Type: TypeUint256,
			},
			wantErr: false,
		},
		{
			name: "valid bool",
			field: Field{
				Name: "myBool",
				Type: TypeBoolean,
			},
			wantErr: false,
		},
		{
			name: "valid boolean",
			field: Field{
				Name: "myBoolean",
				Type: TypeBoolean,
			},
			wantErr: false,
		},
		{
			name: "empty field name",
			field: Field{
				Name: "",
				Type: TypeUint8,
			},
			wantErr: true,
			errMsg:  "field name cannot be empty",
		},
		{
			name: "unknown type",
			field: Field{
				Name: "myField",
				Type: TypeName("unknown"),
			},
			wantErr: true,
			errMsg:  "unknown type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.field.IsValid(nil)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestIsValid_VectorTypes(t *testing.T) {
	tests := []struct {
		name    string
		field   Field
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid vector with size",
			field: Field{
				Name: "myVector",
				Type: TypeVector,
				Size: 32,
			},
			wantErr: false,
		},
		{
			name: "valid bitvector with size",
			field: Field{
				Name: "myBitVector",
				Type: TypeBitVector,
				Size: 256,
			},
			wantErr: false,
		},
		{
			name: "valid bytevector with size",
			field: Field{
				Name: "myByteVector",
				Type: TypeVector,
				Size: 48,
				Children: []Field{
					{Name: "byte", Type: TypeUint8},
				},
			},
			wantErr: false,
		},
		{
			name: "vector without size",
			field: Field{
				Name: "myVector",
				Type: TypeVector,
			},
			wantErr: true,
			errMsg:  "must have non-zero size",
		},
		{
			name: "vector with zero size",
			field: Field{
				Name: "myVector",
				Type: TypeVector,
				Size: 0,
			},
			wantErr: true,
			errMsg:  "must have non-zero size",
		},
		{
			name: "vector with children",
			field: Field{
				Name: "myVector",
				Type: TypeVector,
				Size: 4,
				Children: []Field{
					{Name: "elem", Type: TypeUint32},
				},
			},
			wantErr: false,
		},
		{
			name: "vector with invalid child",
			field: Field{
				Name: "myVector",
				Type: TypeVector,
				Size: 4,
				Children: []Field{
					{Name: "", Type: TypeUint32},
				},
			},
			wantErr: true,
			errMsg:  "field name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.field.IsValid(nil)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestIsValid_ListTypes(t *testing.T) {
	tests := []struct {
		name    string
		field   Field
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid list with limit",
			field: Field{
				Name:  "myList",
				Type:  TypeList,
				Limit: 1024,
			},
			wantErr: false,
		},
		{
			name: "valid bitlist with limit",
			field: Field{
				Name:  "myBitList",
				Type:  TypeBitList,
				Limit: 2048,
			},
			wantErr: false,
		},
		{
			name: "valid bytelist with limit",
			field: Field{
				Name:  "myByteList",
				Type:  TypeList,
				Limit: 65536,
				Children: []Field{
					{Name: "byte", Type: TypeUint8},
				},
			},
			wantErr: false,
		},
		{
			name: "list without limit",
			field: Field{
				Name: "myList",
				Type: TypeList,
			},
			wantErr: true,
			errMsg:  "must have non-zero limit",
		},
		{
			name: "list with zero limit",
			field: Field{
				Name:  "myList",
				Type:  TypeList,
				Limit: 0,
			},
			wantErr: true,
			errMsg:  "must have non-zero limit",
		},
		{
			name: "list with children",
			field: Field{
				Name:  "myList",
				Type:  TypeList,
				Limit: 100,
				Children: []Field{
					{Name: "elem", Type: TypeUint64},
				},
			},
			wantErr: false,
		},
		{
			name: "list with invalid child",
			field: Field{
				Name:  "myList",
				Type:  TypeList,
				Limit: 100,
				Children: []Field{
					{Name: "elem", Type: TypeName("invalid")},
				},
			},
			wantErr: true,
			errMsg:  "unknown type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.field.IsValid(nil)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestIsValid_ContainerType(t *testing.T) {
	tests := []struct {
		name    string
		field   Field
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid container with children",
			field: Field{
				Name: "myContainer",
				Type: TypeContainer,
				Children: []Field{
					{Name: "field1", Type: TypeUint32},
					{Name: "field2", Type: TypeBoolean},
				},
			},
			wantErr: false,
		},
		{
			name: "container without children",
			field: Field{
				Name: "myContainer",
				Type: TypeContainer,
			},
			wantErr: true,
			errMsg:  "must have children",
		},
		{
			name: "container with empty children",
			field: Field{
				Name:     "myContainer",
				Type:     TypeContainer,
				Children: []Field{},
			},
			wantErr: true,
			errMsg:  "must have children",
		},
		{
			name: "container with invalid child",
			field: Field{
				Name: "myContainer",
				Type: TypeContainer,
				Children: []Field{
					{Name: "field1", Type: TypeUint32},
					{Name: "", Type: TypeBoolean},
				},
			},
			wantErr: true,
			errMsg:  "field name cannot be empty",
		},
		{
			name: "nested container",
			field: Field{
				Name: "outerContainer",
				Type: TypeContainer,
				Children: []Field{
					{
						Name: "innerContainer",
						Type: TypeContainer,
						Children: []Field{
							{Name: "value", Type: TypeUint64},
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.field.IsValid(nil)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestIsValid_UnionType(t *testing.T) {
	tests := []struct {
		name    string
		field   Field
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid union with children",
			field: Field{
				Name: "myUnion",
				Type: TypeUnion,
				Children: []Field{
					{Name: "option1", Type: TypeUint32},
					{Name: "option2", Type: TypeBoolean},
				},
			},
			wantErr: false,
		},
		{
			name: "union without children",
			field: Field{
				Name: "myUnion",
				Type: TypeUnion,
			},
			wantErr: true,
			errMsg:  "must have children",
		},
		{
			name: "union with invalid child",
			field: Field{
				Name: "myUnion",
				Type: TypeUnion,
				Children: []Field{
					{Name: "option1", Type: TypeUint32},
					{Name: "option2", Type: TypeName("invalid")},
				},
			},
			wantErr: true,
			errMsg:  "unknown type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.field.IsValid(nil)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestIsValid_RefType(t *testing.T) {
	tests := []struct {
		name    string
		field   Field
		refs    map[string]Field
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid ref",
			field: Field{
				Name: "myRef",
				Type: TypeRef,
				Ref:  "MyType",
			},
			refs: map[string]Field{
				"MyType": {Name: "MyType", Type: TypeUint64},
			},
			wantErr: false,
		},
		{
			name: "ref without reference",
			field: Field{
				Name: "myRef",
				Type: TypeRef,
			},
			refs:    map[string]Field{},
			wantErr: true,
			errMsg:  "no ref specified",
		},
		{
			name: "ref to non-existent type",
			field: Field{
				Name: "myRef",
				Type: TypeRef,
				Ref:  "NonExistent",
			},
			refs:    map[string]Field{},
			wantErr: true,
			errMsg:  "not found",
		},
		{
			name: "ref to invalid type",
			field: Field{
				Name: "myRef",
				Type: TypeRef,
				Ref:  "MyType",
			},
			refs: map[string]Field{
				"MyType": {Name: "", Type: TypeUint64},
			},
			wantErr: true,
			errMsg:  "field name cannot be empty",
		},
		{
			name: "circular reference detection",
			field: Field{
				Name: "myRef",
				Type: TypeRef,
				Ref:  "A",
			},
			refs: map[string]Field{
				"A": {Name: "A", Type: TypeRef, Ref: "B"},
				"B": {Name: "B", Type: TypeRef, Ref: "A"},
			},
			wantErr: true,
			errMsg:  "max iterations reached",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.field.IsValid(tt.refs)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestIsVariable(t *testing.T) {
	tests := []struct {
		name    string
		field   Field
		refs    map[string]Field
		want    bool
		wantErr bool
		errMsg  string
	}{
		{
			name:    "list is variable",
			field:   Field{Name: "myList", Type: TypeList, Limit: 100},
			want:    true,
			wantErr: false,
		},
		{
			name:    "bitlist is variable",
			field:   Field{Name: "myBitList", Type: TypeBitList, Limit: 100},
			want:    true,
			wantErr: false,
		},
		{
			name:    "bytelist is variable",
			field:   Field{Name: "myByteList", Type: TypeList, Limit: 100, Children: []Field{{Name: "byte", Type: TypeUint8}}},
			want:    true,
			wantErr: false,
		},
		{
			name:    "union is variable",
			field:   Field{Name: "myUnion", Type: TypeUnion},
			want:    true,
			wantErr: false,
		},
		{
			name:    "vector is fixed",
			field:   Field{Name: "myVector", Type: TypeVector, Size: 32},
			want:    false,
			wantErr: false,
		},
		{
			name:    "bitvector is fixed",
			field:   Field{Name: "myBitVector", Type: TypeBitVector, Size: 256},
			want:    false,
			wantErr: false,
		},
		{
			name:    "bytevector is fixed",
			field:   Field{Name: "myByteVector", Type: TypeVector, Size: 48, Children: []Field{{Name: "byte", Type: TypeUint8}}},
			want:    false,
			wantErr: false,
		},
		{
			name:    "basic types are fixed",
			field:   Field{Name: "myUint", Type: TypeUint64},
			want:    false,
			wantErr: false,
		},
		{
			name: "container with fixed children is fixed",
			field: Field{
				Name: "myContainer",
				Type: TypeContainer,
				Children: []Field{
					{Name: "field1", Type: TypeUint32},
					{Name: "field2", Type: TypeBoolean},
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "container with variable child is variable",
			field: Field{
				Name: "myContainer",
				Type: TypeContainer,
				Children: []Field{
					{Name: "field1", Type: TypeUint32},
					{Name: "field2", Type: TypeList, Limit: 100},
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "vector with variable children is variable",
			field: Field{
				Name: "myVector",
				Type: TypeVector,
				Size: 4,
				Children: []Field{
					{Name: "elem", Type: TypeList, Limit: 50},
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "ref to fixed type is fixed",
			field: Field{
				Name: "myRef",
				Type: TypeRef,
				Ref:  "FixedType",
			},
			refs: map[string]Field{
				"FixedType": {Name: "FixedType", Type: TypeUint64},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "ref to variable type is variable",
			field: Field{
				Name: "myRef",
				Type: TypeRef,
				Ref:  "VarType",
			},
			refs: map[string]Field{
				"VarType": {Name: "VarType", Type: TypeList, Limit: 100},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "ref without reference",
			field: Field{
				Name: "myRef",
				Type: TypeRef,
			},
			refs:    map[string]Field{},
			want:    false,
			wantErr: true,
			errMsg:  "no ref specified",
		},
		{
			name: "ref to non-existent type",
			field: Field{
				Name: "myRef",
				Type: TypeRef,
				Ref:  "NonExistent",
			},
			refs:    map[string]Field{},
			want:    false,
			wantErr: true,
			errMsg:  "not found",
		},
		{
			name: "circular reference detection",
			field: Field{
				Name: "myRef",
				Type: TypeRef,
				Ref:  "A",
			},
			refs: map[string]Field{
				"A": {Name: "A", Type: TypeRef, Ref: "B"},
				"B": {Name: "B", Type: TypeRef, Ref: "A"},
			},
			want:    false,
			wantErr: true,
			errMsg:  "max iterations reached",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.field.IsVariable(tt.refs)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestComplexScenarios(t *testing.T) {
	t.Run("deeply nested container", func(t *testing.T) {
		field := Field{
			Name: "root",
			Type: TypeContainer,
			Children: []Field{
				{
					Name: "level1",
					Type: TypeContainer,
					Children: []Field{
						{
							Name: "level2",
							Type: TypeContainer,
							Children: []Field{
								{
									Name: "level3",
									Type: TypeVector,
									Size: 4,
									Children: []Field{
										{Name: "elem", Type: TypeUint32},
									},
								},
							},
						},
					},
				},
			},
		}

		err := field.IsValid(nil)
		require.NoError(t, err, "deeply nested container validation failed")

		isVar, err := field.IsVariable(nil)
		require.NoError(t, err)
		assert.False(t, isVar, "deeply nested container should be fixed")
	})

	t.Run("mixed fixed and variable fields", func(t *testing.T) {
		field := Field{
			Name: "mixed",
			Type: TypeContainer,
			Children: []Field{
				{Name: "fixed1", Type: TypeUint32},
				{Name: "fixed2", Type: TypeBitVector, Size: 256},
				{Name: "var1", Type: TypeList, Limit: 100},
				{Name: "fixed3", Type: TypeVector, Size: 48, Children: []Field{{Name: "byte", Type: TypeUint8}}},
				{Name: "var2", Type: TypeBitList, Limit: 2048},
			},
		}

		err := field.IsValid(nil)
		require.NoError(t, err, "mixed container validation failed")

		isVar, err := field.IsVariable(nil)
		require.NoError(t, err)
		assert.True(t, isVar, "container with variable fields should be variable")
	})

	t.Run("ref chain", func(t *testing.T) {
		refs := map[string]Field{
			"Type1": {Name: "Type1", Type: TypeRef, Ref: "Type2"},
			"Type2": {Name: "Type2", Type: TypeRef, Ref: "Type3"},
			"Type3": {Name: "Type3", Type: TypeContainer, Children: []Field{
				{Name: "value", Type: TypeUint64},
			}},
		}

		field := Field{
			Name: "myRef",
			Type: TypeRef,
			Ref:  "Type1",
		}

		err := field.IsValid(refs)
		require.NoError(t, err, "ref chain validation failed")

		isVar, err := field.IsVariable(refs)
		require.NoError(t, err)
		assert.False(t, isVar, "ref chain to fixed type should be fixed")
	})
}

func TestTypeNameConstants(t *testing.T) {
	// Verify all type constants are defined correctly
	expectedTypes := map[TypeName]bool{
		TypeUint8:     true,
		TypeUint16:    true,
		TypeUint32:    true,
		TypeUint64:    true,
		TypeUint128:   true,
		TypeUint256:   true,
		TypeBoolean:   true,
		TypeContainer: true,
		TypeVector:    true,
		TypeList:      true,
		TypeBitVector: true,
		TypeBitList:   true,
		TypeUnion:     true,
		TypeRef:       true,
	}

	// Test string values
	assert.Equal(t, TypeName("uint8"), TypeUint8)
	assert.Equal(t, TypeName("container"), TypeContainer)
	assert.Equal(t, TypeName("ref"), TypeRef)

	// Ensure all expected types are present
	for typeName := range expectedTypes {
		assert.NotEmpty(t, typeName, "Type name should not be empty")
	}
}

