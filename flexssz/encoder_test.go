package flexssz

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBuilder(t *testing.T) {
	t.Run("with writer", func(t *testing.T) {
		buf := new(bytes.Buffer)
		b := NewBuilder(buf)
		assert.NotNil(t, b)
		assert.Equal(t, buf, b.w)
		assert.Nil(t, b.parent)
	})

	t.Run("without writer", func(t *testing.T) {
		b := NewBuilder()
		assert.NotNil(t, b)
		assert.NotNil(t, b.w)
		assert.IsType(t, &bytes.Buffer{}, b.w)
	})
}

func TestBuilder_EncodeUint(t *testing.T) {
	tests := []struct {
		name   string
		encode func(*Builder, any) *Builder
		decode func(*Decoder) (any, error)
		value  any
		size   int
	}{
		{
			name:   "EncodeUint8",
			encode: func(b *Builder, v any) *Builder { return b.EncodeUint8(v.(uint8)) },
			decode: func(d *Decoder) (any, error) { return d.ReadUint8() },
			value:  uint8(0xFF),
			size:   1,
		},
		{
			name:   "EncodeUint16",
			encode: func(b *Builder, v any) *Builder { return b.EncodeUint16(v.(uint16)) },
			decode: func(d *Decoder) (any, error) { return d.ReadUint16() },
			value:  uint16(0xFFFF),
			size:   2,
		},
		{
			name:   "EncodeUint32",
			encode: func(b *Builder, v any) *Builder { return b.EncodeUint32(v.(uint32)) },
			decode: func(d *Decoder) (any, error) { return d.ReadUint32() },
			value:  uint32(0xFFFFFFFF),
			size:   4,
		},
		{
			name:   "EncodeUint64",
			encode: func(b *Builder, v any) *Builder { return b.EncodeUint64(v.(uint64)) },
			decode: func(d *Decoder) (any, error) { return d.ReadUint64() },
			value:  uint64(0xFFFFFFFFFFFFFFFF),
			size:   8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode
			buf := new(bytes.Buffer)
			b := NewBuilder(buf)
			result := tt.encode(b, tt.value)
			assert.Equal(t, b, result) // Check method chaining
			b.Finish()

			// Verify encoded data size
			assert.Equal(t, tt.size, len(buf.Bytes()))

			// Decode and verify
			d := NewDecoder(buf.Bytes())
			decoded, err := tt.decode(d)
			require.NoError(t, err)
			assert.Equal(t, tt.value, decoded)
		})
	}
}

func TestBuilder_EncodeBool(t *testing.T) {
	t.Run("encode true", func(t *testing.T) {
		buf := new(bytes.Buffer)
		b := NewBuilder(buf)

		result := b.EncodeBool(true)
		assert.Equal(t, b, result) // Check method chaining
		b.Finish()

		// Decode and verify
		d := NewDecoder(buf.Bytes())
		decoded, err := d.ReadBool()
		require.NoError(t, err)
		assert.True(t, decoded)
	})

	t.Run("encode false", func(t *testing.T) {
		buf := new(bytes.Buffer)
		b := NewBuilder(buf)

		result := b.EncodeBool(false)
		assert.Equal(t, b, result) // Check method chaining
		b.Finish()

		// Decode and verify
		d := NewDecoder(buf.Bytes())
		decoded, err := d.ReadBool()
		require.NoError(t, err)
		assert.False(t, decoded)
	})
}

func TestBuilder_EncodeFixed(t *testing.T) {
	buf := new(bytes.Buffer)
	b := NewBuilder(buf)

	data := []byte{1, 2, 3, 4, 5}
	result := b.EncodeFixed(data)
	assert.Equal(t, b, result) // Check method chaining
	b.Finish()

	// Decode and verify
	d := NewDecoder(buf.Bytes())
	decoded, err := d.ReadN(5)
	require.NoError(t, err)
	assert.Equal(t, data, decoded)
}

func TestBuilder_EncodeString(t *testing.T) {
	buf := new(bytes.Buffer)
	b := NewBuilder(buf)

	// First encode a fixed element to see proper offset calculation
	b.EncodeUint32(123)
	str := "hello world"
	b.EncodeString(str).Finish()

	// Decode using container operations
	data := buf.Bytes()
	d := NewDecoder(data)

	var fixedVal uint32
	var decodedStr string

	err := d.DecodeContainer(
		Fixed(func(d *Decoder) error {
			return d.ScanUint32(&fixedVal)
		}),
		FixedList(func(d *Decoder) error {
			var err error
			decodedStr, err = d.ReadString()
			return err
		}, 1, len(str)),
	)

	require.NoError(t, err)
	assert.Equal(t, uint32(123), fixedVal)
	assert.Equal(t, str, decodedStr)
}

func TestBuilder_EncodeBytes(t *testing.T) {
	buf := new(bytes.Buffer)
	b := NewBuilder(buf)

	// First encode a fixed element
	b.EncodeUint32(456)
	data := []byte{1, 2, 3, 4, 5}
	b.EncodeBytes(data).Finish()

	// Decode using container operations
	result := buf.Bytes()
	d := NewDecoder(result)

	var fixedVal uint32
	var decodedBytes []byte

	err := d.DecodeContainer(
		Fixed(func(d *Decoder) error {
			return d.ScanUint32(&fixedVal)
		}),
		FixedList(func(d *Decoder) error {
			var err error
			decodedBytes, err = d.ReadBytes()
			return err
		}, 1, len(data)),
	)

	require.NoError(t, err)
	assert.Equal(t, uint32(456), fixedVal)
	assert.Equal(t, data, decodedBytes)
}

func TestBuilder_EncodeBinary(t *testing.T) {
	buf := new(bytes.Buffer)
	b := NewBuilder(buf)

	// Test with various types
	var v1 uint32 = 0x12345678
	var v2 uint64 = 0x123456789ABCDEF0

	b.EncodeBinary(v1).EncodeBinary(v2).Finish()

	// Decode and verify
	d := NewDecoder(buf.Bytes())

	decoded1, err := d.ReadUint32()
	require.NoError(t, err)
	assert.Equal(t, v1, decoded1)

	decoded2, err := d.ReadUint64()
	require.NoError(t, err)
	assert.Equal(t, v2, decoded2)
}

func TestBuilder_Write(t *testing.T) {
	buf := new(bytes.Buffer)
	b := NewBuilder(buf)

	data := []byte{1, 2, 3, 4, 5}
	n, err := b.Write(data)
	require.NoError(t, err)
	assert.Equal(t, len(data), n)

	b.Finish()
	assert.Equal(t, data, buf.Bytes())
}

func TestBuilder_EnterExitDynamic(t *testing.T) {
	t.Run("simple dynamic list", func(t *testing.T) {
		buf := new(bytes.Buffer)
		b := NewBuilder(buf)

		// Encode a fixed value, then a dynamic list
		b.EncodeUint32(42)

		// Enter dynamic context
		dyn := b.EnterDynamic()
		dyn.EncodeUint64(100).EncodeUint64(200).EncodeUint64(300)

		// Exit back to parent
		parent := dyn.ExitDynamic()
		assert.Equal(t, b, parent)

		parent.Finish()

		// Decode and verify
		data := buf.Bytes()
		d := NewDecoder(data)

		var fixedVal uint32
		var dynamicVals []uint64

		err := d.DecodeContainer(
			Fixed(func(d *Decoder) error {
				return d.ScanUint32(&fixedVal)
			}),
			FixedList(func(d *Decoder) error {
				val, err := d.ReadUint64()
				if err != nil {
					return err
				}
				dynamicVals = append(dynamicVals, val)
				return nil
			}, 8, 3),
		)

		require.NoError(t, err)
		assert.Equal(t, uint32(42), fixedVal)
		assert.Equal(t, []uint64{100, 200, 300}, dynamicVals)
	})

	t.Run("nested dynamic lists", func(t *testing.T) {
		buf := new(bytes.Buffer)
		b := NewBuilder(buf)

		b.EncodeUint64(5555)

		// First level dynamic
		dyn1 := b.EnterDynamic()

		// Nested dynamic
		dyn2 := dyn1.EnterDynamic()
		dyn2.EncodeUint64(41).EncodeUint64(42).EncodeUint64(43)
		dyn1 = dyn2.ExitDynamic()

		// Empty dynamic lists
		dyn1.EnterDynamic().ExitDynamic()
		dyn1.EnterDynamic().ExitDynamic()

		// Another nested dynamic
		dyn3 := dyn1.EnterDynamic()
		for i := uint64(41); i <= 48; i++ {
			dyn3.EncodeUint64(i)
		}
		dyn1 = dyn3.ExitDynamic()

		// Exit first level
		b = dyn1.ExitDynamic()
		b.Finish()

		// Decode and verify (matching TestDecodeEncoder2)
		data := buf.Bytes()
		d := NewDecoder(data)

		type S struct {
			v  uint64
			xs [][]uint64
		}
		var s S

		err := d.DecodeContainer([]Op{
			Fixed(func(d *Decoder) error {
				return d.ScanUint64(&s.v)
			}),
			DynamicList(func(d *Decoder) error {
				xs := []uint64{}
				err := d.DecodeFixedList(FixedList(func(d *Decoder) error {
					i, err := d.ReadUint64()
					if err != nil {
						return err
					}
					xs = append(xs, i)
					return nil
				}, 8, 32))
				if err != nil {
					return err
				}
				s.xs = append(s.xs, xs)
				return nil
			}, 32),
		}...)

		require.NoError(t, err)
		assert.Equal(t, uint64(5555), s.v)
		assert.Equal(t, 4, len(s.xs))
		assert.Equal(t, []uint64{41, 42, 43}, s.xs[0])
		assert.Equal(t, []uint64{}, s.xs[1])
		assert.Equal(t, []uint64{}, s.xs[2])
		assert.Equal(t, []uint64{41, 42, 43, 44, 45, 46, 47, 48}, s.xs[3])
	})

	t.Run("dynamic with size guess", func(t *testing.T) {
		buf := new(bytes.Buffer)
		b := NewBuilder(buf)

		// Test with size guess for pre-allocation
		dyn := b.EnterDynamic(100, 200) // Pre-allocate for 300 bytes
		dyn.EncodeUint64(1).EncodeUint64(2)
		b = dyn.ExitDynamic()
		b.Finish()

		assert.NotEmpty(t, buf.Bytes())
	})
}

func TestBuilder_ExitDynamic_Panic(t *testing.T) {
	buf := new(bytes.Buffer)
	b := NewBuilder(buf)

	// Should panic when trying to exit dynamic without entering
	assert.Panics(t, func() {
		b.ExitDynamic()
	})
}

func TestEncodePtr(t *testing.T) {
	tests := []struct {
		offset   int
		expected []byte
	}{
		{0, []byte{0, 0, 0, 0}},
		{100, []byte{100, 0, 0, 0}},
		{0x12345678, []byte{0x78, 0x56, 0x34, 0x12}},
	}

	for _, tt := range tests {
		result := EncodePtr(tt.offset)
		assert.Equal(t, tt.expected, result)
	}
}

func TestBufPtr(t *testing.T) {
	reader := BufPtr(100)
	assert.NotNil(t, reader)

	// Read the data
	data := make([]byte, 4)
	n, err := reader.Read(data)
	require.NoError(t, err)
	assert.Equal(t, 4, n)
	assert.Equal(t, uint32(100), binary.LittleEndian.Uint32(data))
}

func TestValidateBitlist(t *testing.T) {
	tests := []struct {
		name     string
		buf      []byte
		bitLimit uint64
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "empty bitlist",
			buf:      []byte{},
			bitLimit: 100,
			wantErr:  true,
			errMsg:   "bitlist empty",
		},
		{
			name:     "trailing byte is zero",
			buf:      []byte{0xFF, 0x00},
			bitLimit: 100,
			wantErr:  true,
			errMsg:   "trailing byte is zero",
		},
		{
			name:     "too many bytes",
			buf:      []byte{0xFF, 0xFF, 0xFF},
			bitLimit: 10,
			wantErr:  true,
			errMsg:   "unexpected number of bytes",
		},
		{
			name:     "too many bits",
			buf:      []byte{0xFF, 0xFF},
			bitLimit: 10,
			wantErr:  true,
			errMsg:   "too many bits",
		},
		{
			name:     "valid bitlist",
			buf:      []byte{0x01}, // 1 bit set
			bitLimit: 8,
			wantErr:  false,
		},
		{
			name:     "valid bitlist with multiple bytes",
			buf:      []byte{0xFF, 0x01}, // 8 + 0 bits = 8 bits
			bitLimit: 16,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBitlist(tt.buf, tt.bitLimit)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}


func TestBuilder_MethodChaining(t *testing.T) {
	buf := new(bytes.Buffer)
	b := NewBuilder(buf)

	// Test that all methods return the builder for chaining
	result := b.
		EncodeUint8(1).
		EncodeUint16(2).
		EncodeUint32(3).
		EncodeUint64(4).
		EncodeBool(true).
		EncodeFixed([]byte{5, 6}).
		EncodeBinary(uint32(7))

	assert.Equal(t, b, result)

	// Test dynamic chaining
	dyn := b.EnterDynamic()
	dynResult := dyn.
		EncodeUint64(8).
		EncodeString("test").
		EncodeBytes([]byte{9, 10})

	assert.Equal(t, dyn, dynResult)

	parent := dyn.ExitDynamic()
	assert.Equal(t, b, parent)
}

func TestBuilder_EncodeUint256(t *testing.T) {
	t.Run("encode uint128", func(t *testing.T) {
		buf := new(bytes.Buffer)
		b := NewBuilder(buf)

		var val uint256.Int
		val.SetUint64(0xFFFFFFFFFFFFFFFF)
		
		b.EncodeUint128(&val).Finish()

		// Should encode 16 bytes
		data := buf.Bytes()
		assert.Equal(t, 16, len(data))
		
		// First 8 bytes should be 0xFF (little-endian)
		for i := 0; i < 8; i++ {
			assert.Equal(t, uint8(0xFF), data[i])
		}
		// Next 8 bytes should be 0
		for i := 8; i < 16; i++ {
			assert.Equal(t, uint8(0), data[i])
		}
	})

	t.Run("encode uint256", func(t *testing.T) {
		buf := new(bytes.Buffer)
		b := NewBuilder(buf)

		var val uint256.Int
		val.SetFromHex("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF")
		
		b.EncodeUint256(&val).Finish()

		// Should encode 32 bytes
		data := buf.Bytes()
		assert.Equal(t, 32, len(data))
		
		// All bytes should be 0xFF
		for i := 0; i < 32; i++ {
			assert.Equal(t, uint8(0xFF), data[i])
		}
	})

	t.Run("uint256 round trip", func(t *testing.T) {
		buf := new(bytes.Buffer)
		b := NewBuilder(buf)

		var val uint256.Int
		val.SetFromHex("0x123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0")
		
		b.EncodeUint256(&val).Finish()

		// Decode and verify
		data := buf.Bytes()
		d := NewDecoder(data)
		
		// Read back as 4 uint64s
		u0, err := d.ReadUint64()
		require.NoError(t, err)
		u1, err := d.ReadUint64()
		require.NoError(t, err)
		u2, err := d.ReadUint64()
		require.NoError(t, err)
		u3, err := d.ReadUint64()
		require.NoError(t, err)

		// Reconstruct the uint256
		var decoded uint256.Int
		decoded[0] = u0
		decoded[1] = u1
		decoded[2] = u2
		decoded[3] = u3

		assert.Equal(t, val, decoded)
	})
}

func TestEncoderDecoder_RoundTrip(t *testing.T) {
	t.Run("all basic types", func(t *testing.T) {
		buf := new(bytes.Buffer)
		b := NewBuilder(buf)

		// Encode all basic types
		b.EncodeUint8(255).
			EncodeUint16(65535).
			EncodeUint32(4294967295).
			EncodeUint64(18446744073709551615).
			EncodeBool(true).
			EncodeBool(false).
			EncodeFixed([]byte{1, 2, 3, 4}).
			Finish()

		// Decode and verify
		d := NewDecoder(buf.Bytes())

		v1, err := d.ReadUint8()
		require.NoError(t, err)
		assert.Equal(t, uint8(255), v1)

		v2, err := d.ReadUint16()
		require.NoError(t, err)
		assert.Equal(t, uint16(65535), v2)

		v3, err := d.ReadUint32()
		require.NoError(t, err)
		assert.Equal(t, uint32(4294967295), v3)

		v4, err := d.ReadUint64()
		require.NoError(t, err)
		assert.Equal(t, uint64(18446744073709551615), v4)

		b1, err := d.ReadBool()
		require.NoError(t, err)
		assert.True(t, b1)

		b2, err := d.ReadBool()
		require.NoError(t, err)
		assert.False(t, b2)

		fixed, err := d.ReadN(4)
		require.NoError(t, err)
		assert.Equal(t, []byte{1, 2, 3, 4}, fixed)
	})

	t.Run("dynamic structures", func(t *testing.T) {
		buf := new(bytes.Buffer)
		b := NewBuilder(buf)

		// Simpler dynamic structure
		b.EncodeUint32(999).
			EncodeString("test data").
			EnterDynamic().
			EncodeUint64(100).
			EncodeUint64(200).
			EncodeUint64(300).
			ExitDynamic().
			EncodeUint16(777).
			Finish()

		// Decode the structure
		data := buf.Bytes()
		d := NewDecoder(data)

		var fixedVal1 uint32
		var strVal string
		var dynamicVals []uint64
		var fixedVal2 uint16

		// Decode container
		err := d.DecodeContainer(
			Fixed(func(d *Decoder) error {
				return d.ScanUint32(&fixedVal1)
			}),
			FixedList(func(d *Decoder) error {
				var err error
				strVal, err = d.ReadString()
				return err
			}, 1, 100),
			FixedList(func(d *Decoder) error {
				val, err := d.ReadUint64()
				if err != nil {
					return err
				}
				dynamicVals = append(dynamicVals, val)
				return nil
			}, 8, 10),
			Fixed(func(d *Decoder) error {
				return d.ScanUint16(&fixedVal2)
			}),
		)

		require.NoError(t, err)
		assert.Equal(t, uint32(999), fixedVal1)
		assert.Equal(t, "test data", strVal)
		assert.Equal(t, []uint64{100, 200, 300}, dynamicVals)
		assert.Equal(t, uint16(777), fixedVal2)
	})
}

