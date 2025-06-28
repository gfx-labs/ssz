package flexssz

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/holiman/uint256"
)

var order = binary.LittleEndian

type Decoder struct {
	xs  []byte
	cur int
}

func NewDecoder(xs []byte) *Decoder {
	return &Decoder{
		xs: xs,
	}
}

// remaining bytes in buffer, similar to calling buffer.Bytes()
func (d *Decoder) Remaining() []byte {
	return d.xs[d.cur:]
}

// Len returns the total length of the underlying buffer
func (d *Decoder) Len() int {
	return len(d.xs)
}

func (d *Decoder) String() string {
	ans := new(strings.Builder)
	for i, v := range hex.EncodeToString(d.Remaining()) {
		if i%16 == 0 {
			ans.WriteRune('\n')
		}
		ans.WriteRune(v)
	}
	return ans.String()
}

func (d *Decoder) Peek(o []byte) (int, error) {
	if (len(d.xs) - d.cur) < len(o) {
		return 0, fmt.Errorf("ssz: %w", io.ErrUnexpectedEOF)
	}
	n := copy(o, d.xs[d.cur:d.cur+len(o)])
	return n, nil
}

func (d *Decoder) PeekUint32() (i uint32, err error) {
	four := [4]byte{}
	_, err = d.Peek(four[:])
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(four[:]), nil
}
func (d *Decoder) Read(o []byte) (int, error) {
	if d.cur == len(d.xs) {
		return 0, io.EOF
	}
	if (len(d.xs) - d.cur) < len(o) {
		return 0, fmt.Errorf("ssz: %w", io.ErrUnexpectedEOF)
	}
	n := copy(o, d.xs[d.cur:d.cur+len(o)])
	d.cur = d.cur + len(o)
	return n, nil
}

func (d *Decoder) ReadN(n int) ([]byte, error) {
	o := make([]byte, n)
	_, err := d.Read(o)
	return o, err
}
func (d *Decoder) ScanBinary(a any) (err error) {
	err = binary.Read(d, order, a)
	return
}

func (d *Decoder) ScanBinarySlice(a any) (err error) {
	for {
		err = binary.Read(d, order, a)
		if err == io.EOF {
			return nil
		}
		return err
	}
}
func (d *Decoder) ScanUint(i *uint) (err error) {
	err = binary.Read(d, order, i)
	return
}
func (d *Decoder) ScanUint8(i *uint8) (err error) {
	err = binary.Read(d, order, i)
	return
}
func (d *Decoder) ScanUint16(i *uint16) (err error) {
	err = binary.Read(d, order, i)
	return
}
func (d *Decoder) ScanUint32(i *uint32) (err error) {
	err = binary.Read(d, order, i)
	return
}
func (d *Decoder) ScanUint64(i *uint64) (err error) {
	err = binary.Read(d, order, i)
	return
}
func (d *Decoder) ScanBool(a *bool) (err error) {
	ans, err := d.ReadN(1)
	if err != nil {
		return err
	}
	if ans[0] == 1 {
		*a = true
		return nil
	}
	*a = false
	return
}
func (d *Decoder) ReadBool() (b bool, err error) {
	err = d.ScanBool(&b)
	return
}
func (d *Decoder) ReadUint() (i uint, err error) {
	err = d.ScanBinary(&i)
	return
}
func (d *Decoder) ReadUint8() (i uint8, err error) {
	err = d.ScanBinary(&i)
	return
}
func (d *Decoder) ReadUint16() (i uint16, err error) {
	err = d.ScanBinary(&i)
	return
}
func (d *Decoder) ReadUint32() (i uint32, err error) {
	err = d.ScanBinary(&i)
	return
}
func (d *Decoder) ReadUint64() (i uint64, err error) {
	err = d.ScanBinary(&i)
	return
}
func (d *Decoder) ReadOffset() (j int, err error) {
	var i uint32
	err = d.ScanBinary(&i)
	j = int(i)
	return
}

type DecodeFunc func(*Decoder) error

// ContainerElement represents either a fixed or variable field in a container
type ContainerElement struct {
	Fixed    DecodeFunc // Function to decode fixed-size field
	Variable DecodeFunc // Function to decode variable-size field
}

// Fixed creates a ContainerElement for a fixed-size field
func Fixed(fn DecodeFunc) ContainerElement {
	return ContainerElement{Fixed: fn}
}

// Variable creates a ContainerElement for a variable-size field
func Variable(fn DecodeFunc) ContainerElement {
	return ContainerElement{Variable: fn}
}

// Container creates a nested container
func Container(elements ...ContainerElement) DecodeFunc {
	return func(d *Decoder) error {
		return d.DecodeContainer(elements...)
	}
}

// DecodeContainer decodes a container with mixed fixed and variable fields
func (d *Decoder) DecodeContainer(elements ...ContainerElement) error {
	// First pass: read fixed fields and collect offsets
	var offsets []int
	var variableDecoders []DecodeFunc

	for _, elem := range elements {
		if elem.Fixed != nil {
			// Decode fixed field immediately
			if err := elem.Fixed(d); err != nil {
				return err
			}
		} else if elem.Variable != nil {
			// Read offset for variable field
			offset, err := d.ReadOffset()
			if err != nil {
				return err
			}
			offsets = append(offsets, offset)
			variableDecoders = append(variableDecoders, elem.Variable)
		}
	}

	// Second pass: decode variable fields
	for i, decoder := range variableDecoders {
		// Determine the bounds for this field
		start := offsets[i]
		end := d.Len()
		if i+1 < len(offsets) {
			end = offsets[i+1]
		}

		// Validate bounds
		if start > len(d.xs) || end > len(d.xs) || start > end {
			return fmt.Errorf("invalid offset: start=%d, end=%d, len=%d", start, end, len(d.xs))
		}

		// Create decoder for just this field's data
		fieldDecoder := NewDecoder(d.xs[start:end])
		if err := decoder(fieldDecoder); err != nil {
			return err
		}
	}

	return nil
}

// ReadAll reads all remaining bytes in the decoder
func (d *Decoder) ReadAll() ([]byte, error) {
	remaining := d.Remaining()
	if len(remaining) == 0 {
		return []byte{}, nil
	}
	buf := make([]byte, len(remaining))
	n, err := d.Read(buf)
	if err != nil {
		return nil, err
	}
	if n != len(remaining) {
		return nil, io.ErrUnexpectedEOF
	}
	return buf, nil
}

func (d *Decoder) ReadUint128() (*uint256.Int, error) {
	buf := [32]byte{}
	_, err := d.Read(buf[:16])
	if err != nil {
		return nil, err
	}
	val := new(uint256.Int)
	err = val.UnmarshalSSZ(buf[:])
	if err != nil {
		return nil, err
	}

	return val, nil
}

func (d *Decoder) ReadUint256() (*uint256.Int, error) {
	bytes, err := d.ReadN(32)
	if err != nil {
		return nil, err
	}

	val := new(uint256.Int)
	err = val.UnmarshalSSZ(bytes)
	if err != nil {
		return nil, err
	}
	return val, nil
}
