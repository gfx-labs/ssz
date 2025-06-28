package flexssz

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeStruct_VariableFields(t *testing.T) {
	type VariableStruct struct {
		Fixed   uint32   `ssz:"uint32"`
		Name    string   `ssz:"string"`
		Data    []byte   `ssz:"list" ssz-max:"1024"`
		Numbers []uint64 `ssz-max:"10"`
		Fixed2  uint16   `ssz:"uint16"`
	}

	s := VariableStruct{
		Fixed:   999,
		Name:    "test string",
		Data:    []byte{1, 2, 3, 4, 5},
		Numbers: []uint64{100, 200, 300},
		Fixed2:  777,
	}

	encoded, err := EncodeStruct(s)
	require.NoError(t, err)

	// Verify by decoding
	d := NewDecoder(encoded)

	var decoded VariableStruct

	// Decode using container operations
	err = d.DecodeContainer(
		Fixed(func(d *Decoder) error {
			return d.ScanUint32(&decoded.Fixed)
		}),
		FixedList(func(d *Decoder) error {
			var err error
			decoded.Name, err = d.ReadString()
			return err
		}, 1, 100),
		FixedList(func(d *Decoder) error {
			var err error
			decoded.Data, err = d.ReadBytes()
			return err
		}, 1, 100),
		FixedList(func(d *Decoder) error {
			val, err := d.ReadUint64()
			if err != nil {
				return err
			}
			decoded.Numbers = append(decoded.Numbers, val)
			return nil
		}, 8, 10),
		Fixed(func(d *Decoder) error {
			return d.ScanUint16(&decoded.Fixed2)
		}),
	)

	require.NoError(t, err)
	assert.Equal(t, s.Fixed, decoded.Fixed)
	assert.Equal(t, s.Name, decoded.Name)
	assert.Equal(t, s.Data, decoded.Data)
	assert.Equal(t, s.Numbers, decoded.Numbers)
	assert.Equal(t, s.Fixed2, decoded.Fixed2)
}

func TestEncodeStruct_ComplexExample(t *testing.T) {
	type Block struct {
		Slot          uint64   `ssz:"uint64"`
		ProposerIndex uint64   `ssz:"uint64"`
		ParentRoot    [32]byte `ssz:"vector"`
		StateRoot     [32]byte `ssz:"vector"`
		Body          struct {
			RandaoReveal [96]byte `ssz:"vector"`
			Graffiti     [32]byte `ssz:"vector"`
			Deposits     []struct {
				Proof []byte `ssz:"list" ssz-max:"1024"`
				Data  struct {
					Pubkey                [48]byte `ssz:"vector"`
					WithdrawalCredentials [32]byte `ssz:"vector"`
					Amount                uint64   `ssz:"uint64"`
					Signature             [96]byte `ssz:"vector"`
				}
			} `ssz-max:"16"`
		}
	}

	// Create a test block
	block := Block{
		Slot:          12345,
		ProposerIndex: 67,
	}

	// Fill arrays with test data
	for i := range block.ParentRoot {
		block.ParentRoot[i] = byte(i)
	}
	for i := range block.StateRoot {
		block.StateRoot[i] = byte(i + 32)
	}
	for i := range block.Body.RandaoReveal {
		block.Body.RandaoReveal[i] = byte(i % 96)
	}
	for i := range block.Body.Graffiti {
		block.Body.Graffiti[i] = byte(i + 64)
	}

	// Encode
	encoded, err := EncodeStruct(block)
	require.NoError(t, err)
	assert.NotEmpty(t, encoded)

	// Basic verification - check fixed fields at start
	d := NewDecoder(encoded)

	slot, err := d.ReadUint64()
	require.NoError(t, err)
	assert.Equal(t, block.Slot, slot)

	proposer, err := d.ReadUint64()
	require.NoError(t, err)
	assert.Equal(t, block.ProposerIndex, proposer)
}

func TestEncodeStruct_Errors(t *testing.T) {
	t.Run("nil pointer", func(t *testing.T) {
		type TestStruct struct {
			A uint32
		}
		var s *TestStruct
		_, err := EncodeStruct(s)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nil pointer")
	})

	t.Run("not a struct", func(t *testing.T) {
		_, err := EncodeStruct(42)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expected struct")
	})

}

// TestParseSSZTags is removed since parseSSZTags is now unexported.
// The functionality is tested indirectly through PrecacheStructSSZInfo and EncodeStruct.

func TestParseSSZTags_ValidationErrors(t *testing.T) {
	tests := []struct {
		name      string
		field     any
		wantError string
	}{
		{
			name: "uint8 tag on wrong type",
			field: struct {
				F uint16 `ssz:"uint8"`
			}{},
			wantError: "ssz tag 'uint8' requires Go type uint8, got uint16",
		},
		{
			name: "uint16 tag on wrong type",
			field: struct {
				F uint32 `ssz:"uint16"`
			}{},
			wantError: "ssz tag 'uint16' requires Go type uint16, got uint32",
		},
		{
			name: "uint32 tag on wrong type",
			field: struct {
				F string `ssz:"uint32"`
			}{},
			wantError: "ssz tag 'uint32' requires Go type uint32, got string",
		},
		{
			name: "uint64 tag on wrong type",
			field: struct {
				F bool `ssz:"uint64"`
			}{},
			wantError: "ssz tag 'uint64' requires Go type uint64, got bool",
		},
		{
			name: "bool tag on wrong type",
			field: struct {
				F uint8 `ssz:"bool"`
			}{},
			wantError: "ssz tag 'bool' requires Go type bool, got uint8",
		},
		{
			name: "string tag on wrong type",
			field: struct {
				F []byte `ssz:"string"`
			}{},
			wantError: "ssz tag 'string' requires Go type string, got []uint8",
		},
		{
			name: "bytes tag on wrong type",
			field: struct {
				F string `ssz:"list"`
			}{},
			wantError: "ssz tag 'list' requires slice type, got string",
		},
		{
			name: "uint256 tag on wrong type",
			field: struct {
				F uint64 `ssz:"uint256"`
			}{},
			wantError: "ssz tag 'uint256' requires uint256.Int or *uint256.Int type, got uint64",
		},
		{
			name: "uint128 tag on wrong type",
			field: struct {
				F [16]byte `ssz:"uint128"`
			}{},
			wantError: "ssz tag 'uint128' requires uint256.Int or *uint256.Int type, got [16]uint8",
		},
		{
			name: "unsupported type",
			field: struct {
				F chan int
			}{},
			wantError: "unsupported type chan int for SSZ encoding",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := PrecacheStructSSZInfo(tt.field)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantError)
		})
	}
}

func TestPrecacheStructSSZInfo(t *testing.T) {
	type ValidStruct struct {
		A uint32      `ssz:"uint32"`
		B string      `ssz:"string"`
		C uint256.Int `ssz:"uint256"`
		D []byte      `ssz:"list" ssz-max:"2048"`
	}

	// Test successful precaching
	err := PrecacheStructSSZInfo(ValidStruct{})
	require.NoError(t, err)

	// Test with pointer
	err = PrecacheStructSSZInfo(&ValidStruct{})
	require.NoError(t, err)

	// Test with invalid struct
	type InvalidStruct struct {
		A uint32 `ssz:"uint64"` // Type mismatch
	}

	err = PrecacheStructSSZInfo(InvalidStruct{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ssz tag 'uint64' requires Go type uint64")
}

func TestLimitTag(t *testing.T) {
	t.Run("valid slice with limit", func(t *testing.T) {
		type ValidSliceStruct struct {
			Numbers []uint64 `ssz-max:"100"`
			Data    []byte   `ssz:"list" ssz-max:"1024"`
		}

		err := PrecacheStructSSZInfo(ValidSliceStruct{})
		require.NoError(t, err)
	})

	t.Run("slice without limit", func(t *testing.T) {
		type InvalidSliceStruct struct {
			Numbers []uint64 // Missing limit tag
		}

		err := PrecacheStructSSZInfo(InvalidSliceStruct{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "slice types must have either ssz-size or ssz-max tag")
	})

	t.Run("limit on non-slice", func(t *testing.T) {
		type InvalidLimitStruct struct {
			Number uint64 `ssz-max:"100"` // Can't use limit on non-slice
		}

		err := PrecacheStructSSZInfo(InvalidLimitStruct{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ssz-max tag can only be used with slice types")
	})

	t.Run("encode slice with limit", func(t *testing.T) {
		type SliceStruct struct {
			Count  uint32   `ssz:"uint32"`
			Values []uint64 `ssz-max:"10"`
		}

		s := SliceStruct{
			Count:  3,
			Values: []uint64{100, 200, 300},
		}

		encoded, err := EncodeStruct(s)
		require.NoError(t, err)

		// Decode and verify
		d := NewDecoder(encoded)

		var count uint32
		var values []uint64

		err = d.DecodeContainer(
			Fixed(func(d *Decoder) error {
				return d.ScanUint32(&count)
			}),
			FixedList(func(d *Decoder) error {
				val, err := d.ReadUint64()
				if err != nil {
					return err
				}
				values = append(values, val)
				return nil
			}, 8, 10), // Using the limit from the tag
		)

		require.NoError(t, err)
		assert.Equal(t, s.Count, count)
		assert.Equal(t, s.Values, values)
	})

	t.Run("slice exceeds limit", func(t *testing.T) {
		type SliceStruct struct {
			Values []uint64 `ssz-max:"3"`
		}

		s := SliceStruct{
			Values: []uint64{1, 2, 3, 4, 5}, // 5 elements but limit is 3
		}

		_, err := EncodeStruct(s)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "slice length 5 exceeds limit 3")
	})
}

func TestAllSlicesMustHaveLimit(t *testing.T) {
	t.Run("byte slice without limit", func(t *testing.T) {
		type InvalidStruct struct {
			Data []byte `ssz:"list"` // Missing limit tag
		}
		
		err := PrecacheStructSSZInfo(InvalidStruct{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "slice types must have either ssz-size or ssz-max tag")
	})
	
	t.Run("byte slice with limit", func(t *testing.T) {
		type ValidStruct struct {
			Data []byte `ssz:"list" ssz-max:"1024"`
		}
		
		err := PrecacheStructSSZInfo(ValidStruct{})
		require.NoError(t, err)
	})
	
	t.Run("various slice types without limit", func(t *testing.T) {
		testCases := []struct {
			name      string
			structDef interface{}
		}{
			{
				name: "uint64 slice",
				structDef: struct {
					Values []uint64
				}{},
			},
			{
				name: "string slice",
				structDef: struct {
					Names []string
				}{},
			},
			{
				name: "bool slice",
				structDef: struct {
					Flags []bool
				}{},
			},
			{
				name: "struct slice",
				structDef: struct {
					Items []struct {
						ID uint64
					}
				}{},
			},
			{
				name: "byte slice",
				structDef: struct {
					Data []byte
				}{},
			},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := PrecacheStructSSZInfo(tc.structDef)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "slice types must have either ssz-size or ssz-max tag")
			})
		}
	})
	
	t.Run("various slice types with limit", func(t *testing.T) {
		type ValidStruct struct {
			Numbers []uint64 `ssz-max:"100"`
			Names   []string `ssz-max:"50"`
			Flags   []bool   `ssz-max:"200"`
			Data    []byte   `ssz-max:"1024"`
			Items   []struct {
				ID uint64
			} `ssz-max:"10"`
		}
		
		err := PrecacheStructSSZInfo(ValidStruct{})
		require.NoError(t, err)
	})
	
	t.Run("encode byte slice with limit", func(t *testing.T) {
		type ByteSliceStruct struct {
			Header uint32 `ssz:"uint32"`
			Data   []byte `ssz:"list" ssz-max:"100"`
		}
		
		s := ByteSliceStruct{
			Header: 42,
			Data:   []byte{1, 2, 3, 4, 5},
		}
		
		encoded, err := EncodeStruct(s)
		require.NoError(t, err)
		
		// Decode and verify
		d := NewDecoder(encoded)
		
		var header uint32
		var data []byte
		
		err = d.DecodeContainer(
			Fixed(func(d *Decoder) error {
				return d.ScanUint32(&header)
			}),
			FixedList(func(d *Decoder) error {
				var err error
				data, err = d.ReadBytes()
				return err
			}, 1, 100),
		)
		
		require.NoError(t, err)
		assert.Equal(t, s.Header, header)
		assert.Equal(t, s.Data, data)
	})
	
	t.Run("byte slice exceeds limit", func(t *testing.T) {
		type ByteSliceStruct struct {
			Data []byte `ssz:"list" ssz-max:"5"`
		}
		
		s := ByteSliceStruct{
			Data: []byte{1, 2, 3, 4, 5, 6, 7, 8}, // 8 bytes but limit is 5
		}
		
		_, err := EncodeStruct(s)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "slice length 8 exceeds limit 5")
	})
}

