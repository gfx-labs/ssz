package flexssz

import (
	"bytes"
	"encoding/binary"
	"fmt"
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

func TestDecoder_Len(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	d := NewDecoder(data)

	// Len should always return the total buffer size
	assert.Equal(t, 5, d.Len())

	// Read some bytes
	buf := make([]byte, 2)
	_, err := d.Read(buf)
	require.NoError(t, err)

	// Len should still be the same
	assert.Equal(t, 5, d.Len())

	// But remaining should be different
	assert.Equal(t, 3, len(d.Remaining()))
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

func TestDecoder_FixedList_UsingJump(t *testing.T) {
	// Create test data: 4 uint32 values
	var buf bytes.Buffer
	values := []uint32{10, 20, 30, 40}
	for _, v := range values {
		binary.Write(&buf, binary.LittleEndian, v)
	}

	d := NewDecoder(buf.Bytes())

	// Decode the list directly
	var result []uint32
	for i := 0; i < 4; i++ {
		val, err := d.ReadUint32()
		if err != nil {
			break
		}
		result = append(result, val)
	}
	assert.Equal(t, values, result)
}

func TestDecoder_DynamicList_UsingJump(t *testing.T) {
	// Create a dynamic list with 3 items
	var buf bytes.Buffer

	// Write offsets
	binary.Write(&buf, binary.LittleEndian, uint32(12)) // First item at offset 12
	binary.Write(&buf, binary.LittleEndian, uint32(20)) // Second item at offset 20
	binary.Write(&buf, binary.LittleEndian, uint32(28)) // Third item at offset 28

	// Write data
	binary.Write(&buf, binary.LittleEndian, uint64(100)) // First item
	binary.Write(&buf, binary.LittleEndian, uint64(200)) // Second item
	binary.Write(&buf, binary.LittleEndian, uint64(300)) // Third item

	d := NewDecoder(buf.Bytes())

	var result []uint64

	// Read first offset to get count
	firstOffset, err := d.PeekUint32()
	require.NoError(t, err)
	count := firstOffset / 4

	// Read all offsets
	offsets := make([]int, count)
	for i := 0; i < int(count); i++ {
		offset, err := d.ReadOffset()
		require.NoError(t, err)
		offsets[i] = offset
	}

	// Decode each element
	for i, offset := range offsets {
		// Calculate size
		var size int
		if i+1 < len(offsets) {
			size = offsets[i+1] - offset
		} else {
			size = len(buf.Bytes()) - offset
		}

		// Create decoder at offset
		elemDecoder := NewDecoder(buf.Bytes()[offset : offset+size])
		val, err := elemDecoder.ReadUint64()
		require.NoError(t, err)
		result = append(result, val)
	}

	assert.Equal(t, []uint64{100, 200, 300}, result)
}

func TestDecoder_Container_WithDynamicList(t *testing.T) {
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

	// Decode using DecodeContainer
	err := d.DecodeContainer(
		Fixed(func(d *Decoder) error {
			return d.ScanUint64(&fixedVal)
		}),
		Variable(func(d *Decoder) error {

			// Read offset for the dynamic list
			firstOffset, err := d.ReadOffset()
			if err != nil {
				return err
			}
			// the count is the first offset divided by 4
			// since there are N pointers of length 4 bytes before the first offset
			count := firstOffset / 4

			// Read the value
			val, err := d.ReadUint64()
			if err != nil {
				return err
			}
			dynamicVals = append(dynamicVals, val)

			// Verify we read the expected count
			if int(count) != 1 {
				return fmt.Errorf("unexpected count: %d", count)
			}

			return nil
		}),
	)

	require.NoError(t, err)
	assert.Equal(t, uint64(42), fixedVal)
	assert.Equal(t, []uint64{100}, dynamicVals)
}

func TestDecoder_ReadString(t *testing.T) {
	data := []byte("hello world")
	d := NewDecoder(data)

	// Read all remaining bytes
	buf := make([]byte, len(d.Remaining()))
	n, err := d.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, "hello world", string(buf))
	assert.Equal(t, len(data), d.cur)
}

func TestDecoder_ReadBytes(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	d := NewDecoder(data)

	// Read some bytes first
	d.cur = 2

	// Read all remaining bytes
	remaining := d.Remaining()
	buf := make([]byte, len(remaining))
	n, err := d.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, len(remaining), n)
	assert.Equal(t, []byte{3, 4, 5}, buf)
	assert.Equal(t, len(data), d.cur)
}

func TestDecoder_ErrorCases(t *testing.T) {
	t.Run("DecodeContainer with invalid offset", func(t *testing.T) {
		var buf bytes.Buffer
		binary.Write(&buf, binary.LittleEndian, uint32(100)) // Offset beyond data

		d := NewDecoder(buf.Bytes())
		err := d.DecodeContainer(
			Variable(func(d *Decoder) error {
				t.Fatal("Should not reach here")
				return nil
			}),
		)
		assert.Error(t, err)
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
