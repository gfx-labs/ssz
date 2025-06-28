package flexssz

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecoder_Remaining(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	d := NewDecoder(data)

	assert.NotNil(t, d)
	assert.Equal(t, data, d.xs)
	assert.Equal(t, 0, d.cur)
	// Initially all bytes remain
	assert.Equal(t, data, d.Remaining())

	// Read some bytes
	buf := make([]byte, 2)
	_, err := d.Read(buf)
	require.NoError(t, err)

	// Check remaining
	assert.Equal(t, []byte{3, 4, 5}, d.Remaining())
}

func TestDecoder_String(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03, 0x04}
	d := NewDecoder(data)

	str := d.String()
	expected := "\n01020304"
	assert.Equal(t, expected, str)

	// Test with more data to trigger line breaks
	data2 := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11}
	d2 := NewDecoder(data2)
	str2 := d2.String()
	assert.Contains(t, str2, "\n0102030405060708")
	assert.Contains(t, str2, "\n090a0b0c0d0e0f10")
}

func TestDecoder_Peek(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	d := NewDecoder(data)

	// Peek without advancing
	buf := make([]byte, 2)
	n, err := d.Peek(buf)
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	assert.Equal(t, []byte{1, 2}, buf)

	// Cursor should not advance
	assert.Equal(t, 0, d.cur)

	// Peek larger than remaining
	bigBuf := make([]byte, 10)
	_, err = d.Peek(bigBuf)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ssz:")
}

func TestDecoder_PeekUint32(t *testing.T) {
	var buf bytes.Buffer
	val := uint32(0x12345678)
	binary.Write(&buf, binary.LittleEndian, val)

	d := NewDecoder(buf.Bytes())
	peeked, err := d.PeekUint32()
	require.NoError(t, err)
	assert.Equal(t, val, peeked)

	// Cursor should not advance
	assert.Equal(t, 0, d.cur)
}

func TestDecoder_Read(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	d := NewDecoder(data)

	// Normal read
	buf := make([]byte, 2)
	n, err := d.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	assert.Equal(t, []byte{1, 2}, buf)
	assert.Equal(t, 2, d.cur)

	// Read remaining
	buf2 := make([]byte, 3)
	n, err = d.Read(buf2)
	require.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, []byte{3, 4, 5}, buf2)

	// Read at EOF
	buf3 := make([]byte, 1)
	_, err = d.Read(buf3)
	assert.Equal(t, io.EOF, err)

	// Read more than available
	d2 := NewDecoder([]byte{1, 2})
	bigBuf := make([]byte, 5)
	_, err = d2.Read(bigBuf)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ssz:")
}

func TestDecoder_ReadN(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	d := NewDecoder(data)

	// Read N bytes
	result, err := d.ReadN(3)
	require.NoError(t, err)
	assert.Equal(t, []byte{1, 2, 3}, result)

	// Read more than available
	_, err = d.ReadN(10)
	assert.Error(t, err)
}

func TestDecoder_ScanUint(t *testing.T) {
	tests := []struct {
		name string
		fn   func(*Decoder) error
		val  any
		size int
	}{
		{
			name: "uint8",
			fn: func(d *Decoder) error {
				var v uint8
				err := d.ScanUint8(&v)
				assert.Equal(t, uint8(0x12), v)
				return err
			},
			val:  uint8(0x12),
			size: 1,
		},
		{
			name: "uint16",
			fn: func(d *Decoder) error {
				var v uint16
				err := d.ScanUint16(&v)
				assert.Equal(t, uint16(0x1234), v)
				return err
			},
			val:  uint16(0x1234),
			size: 2,
		},
		{
			name: "uint32",
			fn: func(d *Decoder) error {
				var v uint32
				err := d.ScanUint32(&v)
				assert.Equal(t, uint32(0x12345678), v)
				return err
			},
			val:  uint32(0x12345678),
			size: 4,
		},
		{
			name: "uint64",
			fn: func(d *Decoder) error {
				var v uint64
				err := d.ScanUint64(&v)
				assert.Equal(t, uint64(0x123456789abcdef0), v)
				return err
			},
			val:  uint64(0x123456789abcdef0),
			size: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			binary.Write(&buf, binary.LittleEndian, tt.val)
			d := NewDecoder(buf.Bytes())

			err := tt.fn(d)
			require.NoError(t, err)
			assert.Equal(t, tt.size, d.cur)
		})
	}
}

func TestDecoder_ScanBool(t *testing.T) {
	// Test true
	d1 := NewDecoder([]byte{1})
	var b1 bool
	err := d1.ScanBool(&b1)
	require.NoError(t, err)
	assert.True(t, b1)

	// Test false
	d2 := NewDecoder([]byte{0})
	var b2 bool
	err = d2.ScanBool(&b2)
	require.NoError(t, err)
	assert.False(t, b2)

	// Test any non-1 value is false
	d3 := NewDecoder([]byte{2})
	var b3 bool
	err = d3.ScanBool(&b3)
	require.NoError(t, err)
	assert.False(t, b3)
}

func TestDecoder_ReadUint(t *testing.T) {
	tests := []struct {
		name   string
		fn     func(*Decoder) (any, error)
		val    any
		encode func(any) []byte
	}{
		{
			name: "ReadBool true",
			fn: func(d *Decoder) (any, error) {
				return d.ReadBool()
			},
			val:    true,
			encode: func(v any) []byte { return []byte{1} },
		},
		{
			name: "ReadBool false",
			fn: func(d *Decoder) (any, error) {
				return d.ReadBool()
			},
			val:    false,
			encode: func(v any) []byte { return []byte{0} },
		},
		{
			name: "ReadUint8",
			fn: func(d *Decoder) (any, error) {
				return d.ReadUint8()
			},
			val: uint8(0xFF),
			encode: func(v any) []byte {
				var buf bytes.Buffer
				binary.Write(&buf, binary.LittleEndian, v)
				return buf.Bytes()
			},
		},
		{
			name: "ReadUint16",
			fn: func(d *Decoder) (any, error) {
				return d.ReadUint16()
			},
			val: uint16(0xFFFF),
			encode: func(v any) []byte {
				var buf bytes.Buffer
				binary.Write(&buf, binary.LittleEndian, v)
				return buf.Bytes()
			},
		},
		{
			name: "ReadUint32",
			fn: func(d *Decoder) (any, error) {
				return d.ReadUint32()
			},
			val: uint32(0xFFFFFFFF),
			encode: func(v any) []byte {
				var buf bytes.Buffer
				binary.Write(&buf, binary.LittleEndian, v)
				return buf.Bytes()
			},
		},
		{
			name: "ReadUint64",
			fn: func(d *Decoder) (any, error) {
				return d.ReadUint64()
			},
			val: uint64(0xFFFFFFFFFFFFFFFF),
			encode: func(v any) []byte {
				var buf bytes.Buffer
				binary.Write(&buf, binary.LittleEndian, v)
				return buf.Bytes()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := tt.encode(tt.val)
			d := NewDecoder(data)

			result, err := tt.fn(d)
			require.NoError(t, err)
			assert.Equal(t, tt.val, result)
		})
	}
}

func TestDecoder_ReadOffset(t *testing.T) {
	var buf bytes.Buffer
	offset := uint32(100)
	binary.Write(&buf, binary.LittleEndian, offset)

	d := NewDecoder(buf.Bytes())
	result, err := d.ReadOffset()
	require.NoError(t, err)
	assert.Equal(t, int(offset), result)
}

func TestDecoder_DecodeFixedList(t *testing.T) {
	// Create test data: 4 uint32 values
	var buf bytes.Buffer
	values := []uint32{10, 20, 30, 40}
	for _, v := range values {
		binary.Write(&buf, binary.LittleEndian, v)
	}

	d := NewDecoder(buf.Bytes())

	// Decode the list
	var result []uint32
	op := FixedList(func(d *Decoder) error {
		val, err := d.ReadUint32()
		if err != nil {
			return err
		}
		result = append(result, val)
		return nil
	}, 4, 10) // chunk size 4 bytes, max 10 chunks

	err := d.DecodeFixedList(op)
	require.NoError(t, err)
	assert.Equal(t, values, result)
}

func TestDecoder_DecodeFixedList_Errors(t *testing.T) {
	t.Run("exceeds max size", func(t *testing.T) {
		// Create data that exceeds max
		data := make([]byte, 100)
		d := NewDecoder(data)

		op := FixedList(func(d *Decoder) error {
			_, err := d.ReadUint32()
			return err
		}, 4, 2) // Only allow 2 chunks = 8 bytes

		err := d.DecodeFixedList(op)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "flist big")
	})

	t.Run("extra bytes", func(t *testing.T) {
		// Create data with extra bytes
		data := []byte{1, 2, 3, 4, 5} // 5 bytes, not divisible by 4
		d := NewDecoder(data)

		op := FixedList(func(d *Decoder) error {
			_, err := d.ReadUint32()
			return err
		}, 4, 10)

		err := d.DecodeFixedList(op)
		assert.Error(t, err)
	})
}

func TestDecoder_DecodeDynamicList(t *testing.T) {
	// Create a dynamic list with 3 items
	var buf bytes.Buffer

	// Write offsets
	binary.Write(&buf, binary.LittleEndian, uint32(12)) // First item at offset 12 (after 3*4 bytes of offsets)
	binary.Write(&buf, binary.LittleEndian, uint32(20)) // Second item at offset 20
	binary.Write(&buf, binary.LittleEndian, uint32(28)) // Third item at offset 28

	// Write data
	binary.Write(&buf, binary.LittleEndian, uint64(100)) // First item
	binary.Write(&buf, binary.LittleEndian, uint64(200)) // Second item
	binary.Write(&buf, binary.LittleEndian, uint64(300)) // Third item

	d := NewDecoder(buf.Bytes())

	var result []uint64
	op := DynamicList(func(d *Decoder) error {
		val, err := d.ReadUint64()
		if err != nil {
			return err
		}
		result = append(result, val)
		return nil
	}, 10)

	err := d.DecodeDynamicList(op)
	require.NoError(t, err)
	assert.Equal(t, []uint64{100, 200, 300}, result)
}

func TestDecoder_DecodeContainer(t *testing.T) {
	// Create a container with fixed and dynamic elements
	var buf bytes.Buffer

	// Fixed element: uint64
	binary.Write(&buf, binary.LittleEndian, uint64(42))

	// Offset for dynamic list
	binary.Write(&buf, binary.LittleEndian, uint32(12)) // Dynamic list starts at offset 12

	// Dynamic list data
	binary.Write(&buf, binary.LittleEndian, uint32(4))   // One offset for one item
	binary.Write(&buf, binary.LittleEndian, uint64(100)) // The item

	d := NewDecoder(buf.Bytes())

	var fixedVal uint64
	var dynamicVals []uint64

	err := d.DecodeContainer(
		Fixed(func(d *Decoder) error {
			return d.ScanUint64(&fixedVal)
		}),
		DynamicList(func(d *Decoder) error {
			val, err := d.ReadUint64()
			if err != nil {
				return err
			}
			dynamicVals = append(dynamicVals, val)
			return nil
		}, 10),
	)

	require.NoError(t, err)
	assert.Equal(t, uint64(42), fixedVal)
	assert.Equal(t, []uint64{100}, dynamicVals)
}

func TestDecoder_Jump(t *testing.T) {
	// Create data with an offset
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint32(8)) // Offset to position 8
	buf.Write([]byte{0, 0, 0, 0})                      // Padding
	buf.Write([]byte{1, 2, 3, 4})                      // Data at offset 8

	d := NewDecoder(buf.Bytes())

	jumped, offset, err := d.Jump()
	require.NoError(t, err)
	assert.Equal(t, 8, offset)
	assert.Equal(t, []byte{1, 2, 3, 4}, jumped.Remaining())
}

func TestDecoder_JumpLen(t *testing.T) {
	// JumpLen returns the offset divided by 4
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint32(16)) // Offset 16
	// Add padding data to make the jump valid
	buf.Write(make([]byte, 12)) // 4 bytes for offset + 12 = 16 total

	d := NewDecoder(buf.Bytes())

	dec, length, err := d.JumpLen()
	require.NoError(t, err)
	assert.Equal(t, 4, length) // 16/4 = 4
	assert.NotNil(t, dec)
}

func TestDecoder_ReadString(t *testing.T) {
	data := []byte("hello world")
	d := NewDecoder(data)

	str, err := d.ReadString()
	require.NoError(t, err)
	assert.Equal(t, "hello world", str)
	assert.Equal(t, len(data), d.cur)
}

func TestDecoder_ReadBytes(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	d := NewDecoder(data)

	// Read some bytes first
	d.cur = 2

	result, err := d.ReadBytes()
	require.NoError(t, err)
	assert.Equal(t, []byte{3, 4, 5}, result)
	assert.Equal(t, len(data), d.cur)
}

func TestDecoder_ErrorCases(t *testing.T) {
	t.Run("Jump with invalid offset", func(t *testing.T) {
		var buf bytes.Buffer
		binary.Write(&buf, binary.LittleEndian, uint32(100)) // Offset beyond data

		d := NewDecoder(buf.Bytes())
		_, _, err := d.Jump()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "dynamic overflow")
	})

	t.Run("DecodeContainer with invalid opcode", func(t *testing.T) {
		d := NewDecoder([]byte{})
		err := d.DecodeContainer(Op{O: 999}) // Invalid opcode
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "opcode")
	})

	t.Run("DecodeDynamicList with count too large", func(t *testing.T) {
		var buf bytes.Buffer
		binary.Write(&buf, binary.LittleEndian, uint32(40)) // 10 items

		d := NewDecoder(buf.Bytes())
		op := DynamicList(func(d *Decoder) error { return nil }, 5) // Max 5

		err := d.DecodeDynamicList(op)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "dlist big")
	})
}

func TestDecoder_ScanBinarySlice(t *testing.T) {
	var buf bytes.Buffer
	values := []uint32{10, 20, 30}
	for _, v := range values {
		binary.Write(&buf, binary.LittleEndian, v)
	}

	d := NewDecoder(buf.Bytes())

	// This should read until EOF
	var val uint32
	err := d.ScanBinarySlice(&val)
	require.NoError(t, err)
	assert.Equal(t, uint32(10), val)
	assert.Equal(t, 4, d.cur)
}

func TestDecoder_PeekBinary(t *testing.T) {
	var buf bytes.Buffer
	val := uint64(0x123456789abcdef0)
	binary.Write(&buf, binary.LittleEndian, val)

	d := NewDecoder(buf.Bytes())

	var peeked uint64
	err := d.PeekBinary(&peeked)
	require.NoError(t, err)
	assert.Equal(t, val, peeked)
	assert.Equal(t, 0, d.cur) // Cursor should not advance
}

