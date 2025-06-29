package flexssz

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshal_EmptySlices(t *testing.T) {
	type SliceStruct struct {
		Before  uint32   `ssz:"uint32"`
		Empty   []byte   `ssz:"list" ssz-max:"100"`
		Numbers []uint64 `ssz-max:"10"`
		After   uint32   `ssz:"uint32"`
	}

	// Test with empty slices
	original := SliceStruct{
		Before:  123,
		Empty:   []byte{},
		Numbers: []uint64{},
		After:   456,
	}

	encoded, err := Marshal(original)
	require.NoError(t, err)

	var decoded SliceStruct
	err = Unmarshal(encoded, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original, decoded)
}

func TestUnmarshal_SkipFields(t *testing.T) {
	type SkipStruct struct {
		Include1 uint32 `ssz:"uint32"`
		Skip1    string `ssz:"-"`
		Include2 bool   `ssz:"bool"`
		skip2    uint64 // unexported
		Include3 uint16 `ssz:"uint16"`
	}

	// Encode
	original := SkipStruct{
		Include1: 100,
		Skip1:    "ignored",
		Include2: true,
		skip2:    999999,
		Include3: 200,
	}

	encoded, err := Marshal(original)
	require.NoError(t, err)

	// Decode
	var decoded SkipStruct
	decoded.Skip1 = "different" // Should not be overwritten
	decoded.skip2 = 777777      // Should not be overwritten
	
	err = Unmarshal(encoded, &decoded)
	require.NoError(t, err)

	// Check that only non-skipped fields were decoded
	assert.Equal(t, original.Include1, decoded.Include1)
	assert.Equal(t, original.Include2, decoded.Include2)
	assert.Equal(t, original.Include3, decoded.Include3)
	assert.Equal(t, "different", decoded.Skip1) // Unchanged
	assert.Equal(t, uint64(777777), decoded.skip2) // Unchanged
}

func TestUnmarshal_ComplexExample(t *testing.T) {
	type Deposit struct {
		Proof []byte `ssz:"list" ssz-max:"1024"`
		Data  struct {
			Pubkey                [48]byte `ssz:"vector"`
			WithdrawalCredentials [32]byte `ssz:"vector"`
			Amount                uint64   `ssz:"uint64"`
			Signature             [96]byte `ssz:"vector"`
		}
	}
	
	type Block struct {
		Slot          uint64   `ssz:"uint64"`
		ProposerIndex uint64   `ssz:"uint64"`
		ParentRoot    [32]byte `ssz:"vector"`
		StateRoot     [32]byte `ssz:"vector"`
		Body          struct {
			RandaoReveal [96]byte  `ssz:"vector"`
			Graffiti     [32]byte  `ssz:"vector"`
			Deposits     []Deposit `ssz-max:"16"`
		}
	}

	// Create original
	original := Block{
		Slot:          12345,
		ProposerIndex: 67,
	}

	// Fill arrays
	for i := range original.ParentRoot {
		original.ParentRoot[i] = byte(i)
	}
	for i := range original.StateRoot {
		original.StateRoot[i] = byte(i + 32)
	}
	for i := range original.Body.RandaoReveal {
		original.Body.RandaoReveal[i] = byte(i % 96)
	}
	for i := range original.Body.Graffiti {
		original.Body.Graffiti[i] = byte(i + 64)
	}

	// Add deposits
	original.Body.Deposits = []Deposit{
		{
			Proof: []byte{1, 2, 3, 4, 5},
			Data: struct {
				Pubkey                [48]byte `ssz:"vector"`
				WithdrawalCredentials [32]byte `ssz:"vector"`
				Amount                uint64   `ssz:"uint64"`
				Signature             [96]byte `ssz:"vector"`
			}{
				Amount: 32000000000,
			},
		},
		{
			Proof: []byte{6, 7, 8},
			Data: struct {
				Pubkey                [48]byte `ssz:"vector"`
				WithdrawalCredentials [32]byte `ssz:"vector"`
				Amount                uint64   `ssz:"uint64"`
				Signature             [96]byte `ssz:"vector"`
			}{
				Amount: 16000000000,
			},
		},
	}

	// Fill deposit arrays
	for i := range original.Body.Deposits[0].Data.Pubkey {
		original.Body.Deposits[0].Data.Pubkey[i] = byte(i)
	}
	for i := range original.Body.Deposits[0].Data.WithdrawalCredentials {
		original.Body.Deposits[0].Data.WithdrawalCredentials[i] = byte(i + 48)
	}
	for i := range original.Body.Deposits[0].Data.Signature {
		original.Body.Deposits[0].Data.Signature[i] = byte(i % 96)
	}

	// Encode
	encoded, err := Marshal(original)
	require.NoError(t, err)

	// Decode
	var decoded Block
	err = Unmarshal(encoded, &decoded)
	require.NoError(t, err)

	// Compare
	assert.Equal(t, original.Slot, decoded.Slot)
	assert.Equal(t, original.ProposerIndex, decoded.ProposerIndex)
	assert.Equal(t, original.ParentRoot, decoded.ParentRoot)
	assert.Equal(t, original.StateRoot, decoded.StateRoot)
	assert.Equal(t, original.Body.RandaoReveal, decoded.Body.RandaoReveal)
	assert.Equal(t, original.Body.Graffiti, decoded.Body.Graffiti)
	assert.Equal(t, len(original.Body.Deposits), len(decoded.Body.Deposits))
	
	for i, dep := range original.Body.Deposits {
		assert.Equal(t, dep.Proof, decoded.Body.Deposits[i].Proof)
		assert.Equal(t, dep.Data, decoded.Body.Deposits[i].Data)
	}
}

func TestUnmarshal_Uint256Pointers(t *testing.T) {
	t.Run("decode struct with uint256 pointer", func(t *testing.T) {
		type WithPointer struct {
			Value *uint256.Int `ssz:"uint256"`
		}

		// Encode
		val := uint256.NewInt(0x123456789ABCDEF0)
		original := WithPointer{
			Value: val,
		}

		encoded, err := Marshal(original)
		require.NoError(t, err)

		// Decode
		var decoded WithPointer
		err = Unmarshal(encoded, &decoded)
		require.NoError(t, err)

		// Check pointer is allocated
		require.NotNil(t, decoded.Value)
		assert.Equal(t, original.Value.String(), decoded.Value.String())
	})

	t.Run("decode struct with uint128 pointer", func(t *testing.T) {
		type WithPointer struct {
			Value *uint256.Int `ssz:"uint128"`
		}

		// Encode
		val := uint256.NewInt(0x123456789ABCDEF0)
		original := WithPointer{
			Value: val,
		}

		encoded, err := Marshal(original)
		require.NoError(t, err)

		// Decode
		var decoded WithPointer
		err = Unmarshal(encoded, &decoded)
		require.NoError(t, err)

		// Check pointer is allocated
		require.NotNil(t, decoded.Value)
		assert.Equal(t, original.Value.String(), decoded.Value.String())
	})

	t.Run("mixed value and pointer types", func(t *testing.T) {
		type Mixed struct {
			DirectValue  uint256.Int  `ssz:"uint256"`
			PointerValue *uint256.Int `ssz:"uint256"`
			Uint128Ptr   *uint256.Int `ssz:"uint128"`
		}

		// Encode
		val1 := uint256.NewInt(100)
		val2 := uint256.NewInt(200)
		val3 := uint256.NewInt(300)

		original := Mixed{
			DirectValue:  *val1,
			PointerValue: val2,
			Uint128Ptr:   val3,
		}

		encoded, err := Marshal(original)
		require.NoError(t, err)

		// Decode
		var decoded Mixed
		err = Unmarshal(encoded, &decoded)
		require.NoError(t, err)

		// Compare
		assert.Equal(t, original.DirectValue.String(), decoded.DirectValue.String())
		require.NotNil(t, decoded.PointerValue)
		assert.Equal(t, original.PointerValue.String(), decoded.PointerValue.String())
		require.NotNil(t, decoded.Uint128Ptr)
		assert.Equal(t, original.Uint128Ptr.String(), decoded.Uint128Ptr.String())
	})
}

func TestUnmarshal_Errors(t *testing.T) {
	t.Run("nil pointer", func(t *testing.T) {
		type TestStruct struct {
			A uint32 `ssz:"uint32"`
		}

		data := []byte{1, 2, 3, 4}
		var s *TestStruct
		err := Unmarshal(data, s)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must not be nil")
	})

	t.Run("not a pointer", func(t *testing.T) {
		type TestStruct struct {
			A uint32 `ssz:"uint32"`
		}

		data := []byte{1, 2, 3, 4}
		var s TestStruct
		err := Unmarshal(data, s)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be a pointer")
	})

	t.Run("not a struct", func(t *testing.T) {
		// This test is no longer valid since Unmarshal now works with any type
		// Test that uint32 unmarshal works correctly instead
		data := []byte{1, 2, 3, 4}
		var s uint32
		err := Unmarshal(data, &s)
		assert.NoError(t, err)
		assert.Equal(t, uint32(0x04030201), s) // Little-endian
	})

	t.Run("insufficient data", func(t *testing.T) {
		type TestStruct struct {
			A uint32 `ssz:"uint32"`
			B uint64 `ssz:"uint64"`
		}

		data := []byte{1, 2, 3} // Not enough bytes
		var s TestStruct
		err := Unmarshal(data, &s)
		assert.Error(t, err)
	})

	t.Run("slice exceeds limit", func(t *testing.T) {
		type TestStruct struct {
			Data []byte `ssz:"list" ssz-max:"5"`
		}

		// We're not using the properly encoded data here since we want to test
		// what happens when the decoder encounters data that exceeds the limit

		// Manually modify the encoded data to have more bytes
		// First 4 bytes are the offset (8 for uint32 offset)
		// Replace the data portion with too many bytes
		badEncoded := make([]byte, 4+10) // 4 byte offset + 10 bytes of data (exceeds limit of 5)
		badEncoded[0] = 4 // Offset = 4
		for i := 4; i < len(badEncoded); i++ {
			badEncoded[i] = byte(i)
		}

		var decoded TestStruct
		err := Unmarshal(badEncoded, &decoded)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds limit")
	})
}

func TestUnmarshal_SliceOfStructs(t *testing.T) {
	type Inner struct {
		ID    uint32 `ssz:"uint32"`
		Value uint64 `ssz:"uint64"`
		Name  string `ssz:"string"`
	}

	type Outer struct {
		Count uint32  `ssz:"uint32"`
		Items []Inner `ssz-max:"10"`
	}

	// Encode
	original := Outer{
		Count: 3,
		Items: []Inner{
			{ID: 1, Value: 100, Name: "first"},
			{ID: 2, Value: 200, Name: "second"},
			{ID: 3, Value: 300, Name: "third"},
		},
	}

	encoded, err := Marshal(original)
	require.NoError(t, err)

	// Decode
	var decoded Outer
	err = Unmarshal(encoded, &decoded)
	require.NoError(t, err)

	// Compare
	assert.Equal(t, original, decoded)
}

func TestUnmarshal_MixedFixedVariable(t *testing.T) {
	type Complex struct {
		// Fixed fields
		A uint8    `ssz:"uint8"`
		B uint16   `ssz:"uint16"`
		C [4]byte  `ssz:"vector"`
		
		// Variable field
		D string   `ssz:"string"`
		
		// More fixed fields
		E uint32   `ssz:"uint32"`
		F bool     `ssz:"bool"`
		
		// More variable fields
		G []uint64 `ssz-max:"5"`
		H []byte   `ssz:"list" ssz-max:"100"`
		
		// Final fixed field
		I uint64   `ssz:"uint64"`
	}

	// Encode
	original := Complex{
		A: 123,
		B: 45678,
		C: [4]byte{1, 2, 3, 4},
		D: "hello world",
		E: 87654321,
		F: true,
		G: []uint64{111, 222, 333},
		H: []byte("test data"),
		I: 9876543210,
	}

	encoded, err := Marshal(original)
	require.NoError(t, err)

	// Decode
	var decoded Complex
	err = Unmarshal(encoded, &decoded)
	require.NoError(t, err)

	// Compare
	assert.Equal(t, original, decoded)
}