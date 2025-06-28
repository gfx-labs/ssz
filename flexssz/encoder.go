package flexssz

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math/bits"

	"github.com/holiman/uint256"
)

/*
ssz can be abstracted using a stack + heap
the stack may contain a "virtual pointer" to the heap, or simply data
since it is impossible for a variable-size offset to start from -1, we may use that value of the pointer as a check to see if it is a pointer
*/

const PtrSize = 4

// the word, which contains either pointer or bytes data
type word struct {
	pointer int
	//dat     []byte
	dat EncodeFunc
}

// either encodes dat if pointer is 0, or the 4 bytes of word
func (v *word) EncodeTo(plus int, w io.Writer) error {
	if v.pointer < 0 {
		return v.dat(w)
	}
	return binary.Write(w, order, uint32(plus+v.pointer))
}

type memory struct {
	stack       []word
	currentHeap EncodeFunc
	// current stack pointer
	cur uint32
	// current heap size
	hz int
}

func (m *Builder) Write(xs []byte) (int, error) {
	m.stack = append(m.stack, word{
		pointer: -1,
		//	dat:     xs,
		dat: func(w io.Writer) error {
			_, err := w.Write(xs)
			return err
		},
	})
	// increase current cursor by amount of data written to stack
	m.cur = m.cur + uint32(len(xs))
	return len(xs), nil
}

func (m *Builder) appendHeap(sz int, r EncodeFunc) {
	// advance by the size of the pointer
	m.cur = m.cur + PtrSize
	// the current pointer is at the heap. we will add the stack size later
	m.stack = append(m.stack, word{
		pointer: m.hz,
	})
	// now advance the heap cursor
	m.hz = m.hz + sz
	// and append to heap
	m.currentHeap = curryHeap(m.currentHeap, r)
}

func curryHeap(curf, r EncodeFunc) EncodeFunc {
	return func(w io.Writer) error {
		if curf != nil {
			err := curf(w)
			if err != nil {
				return err
			}
		}
		if r != nil {
			err := r(w)
			if err != nil {
				return err
			}
		}
		return nil
	}
}

type EncodeFunc = func(io.Writer) error

func WriteStaticList[T any](d *Builder, xs []T) EncodeFunc {
	return nil
}

func (d *Builder) EnterDynamic(guess ...int) *Builder {
	b := &Builder{parent: d}
	sz := 0
	for _, v := range guess {
		sz = sz + v
	}
	b.stack = make([]word, 0, sz)
	return b
}
func (d *Builder) ExitDynamic() *Builder {
	if d.parent == nil {
		panic("tried to exit variable context when not in one")
	}
	d.parent.appendHeap(d.hz+int(d.cur), func(w io.Writer) (err error) {
		for _, v := range d.stack {
			err = v.EncodeTo(int(d.cur), w)
			if err != nil {
				return err
			}
		}
		if d.currentHeap != nil {
			err = d.currentHeap(w)
			if err != nil {
				return err
			}
		}
		return nil
	})
	return d.parent
}

func (d *Builder) Finish() error {
	for _, v := range d.stack {
		err := v.EncodeTo(int(d.cur), d.w)
		if err != nil {
			return err
		}
	}
	if d.currentHeap != nil {
		err := d.currentHeap(d.w)
		if err != nil {
			return err
		}
	}
	return nil
}

func EncodePtr(i int) []byte {
	bin := make([]byte, 4)
	order.PutUint32(bin, uint32(i))
	return bin
}
func BufPtr(i int) io.Reader {
	return bytes.NewBuffer(EncodePtr(i))
}

func (d *Builder) EncodeBytes(xs []byte) *Builder {
	d.appendHeap(len(xs), func(b io.Writer) (err error) {
		_, err = b.Write(xs)
		return
	})
	return d
}

func (d *Builder) EncodeString(s string) *Builder {
	//TODO: optimize
	return d.EncodeBytes([]byte(s))
}

type Builder struct {
	parent *Builder
	w      io.Writer

	memory
}

func NewBuilder(w ...io.Writer) *Builder {
	b := &Builder{}
	if len(w) == 0 {
		b.w = new(bytes.Buffer)
	} else {
		b.w = w[0]
	}
	return b
}

func (d *Builder) EncodeBool(b bool) *Builder {
	if b == true {
		d.Write([]byte{0x1})
		return d
	}
	d.Write([]byte{0x0})
	return d
}
func (d *Builder) EncodeUint8(i uint8) *Builder {
	return d.EncodeBinary(i)
}
func (d *Builder) EncodeUint16(i uint16) *Builder {
	return d.EncodeBinary(i)
}
func (d *Builder) EncodeUint32(i uint32) *Builder {
	return d.EncodeBinary(i)
}
func (d *Builder) EncodeUint64(i uint64) *Builder {
	return d.EncodeBinary(i)
}
func (d *Builder) EncodeUint128(i *uint256.Int) *Builder {
	// uint128 uses the lower 2 uint64s (16 bytes)
	// uint256.Int is [4]uint64 in little-endian order
	d.EncodeUint64(i[0])
	d.EncodeUint64(i[1])
	return d
}
func (d *Builder) EncodeUint256(i *uint256.Int) *Builder {
	// uint256 uses all 4 uint64s (32 bytes)
	// uint256.Int is [4]uint64 in little-endian order
	d.EncodeUint64(i[0])
	d.EncodeUint64(i[1])
	d.EncodeUint64(i[2])
	d.EncodeUint64(i[3])
	return d
}
func (d *Builder) EncodeBinary(i any) *Builder {
	binary.Write(d, order, i)
	return d
}
func (d *Builder) EncodeFixed(val []byte) *Builder {
	d.Write(val)
	return d
}

// from fastssz
func ValidateBitlist(buf []byte, bitLimit uint64) error {
	byteLen := len(buf)
	if byteLen == 0 {
		return fmt.Errorf("bitlist empty, it does not have length bit")
	}
	// Maximum possible bytes in a bitlist with provided bitlimit.
	maxBytes := (bitLimit >> 3) + 1
	if byteLen > int(maxBytes) {
		return fmt.Errorf("unexpected number of bytes, got %d but found %d", byteLen, maxBytes)
	}
	// The most significant bit is present in the last byte in the array.
	last := buf[byteLen-1]
	if last == 0 {
		return fmt.Errorf("trailing byte is zero")
	}
	// Determine the position of the most significant bit.
	msb := bits.Len8(last)
	// The absolute position of the most significant bit will be the number of
	// bits in the preceding bytes plus the position of the most significant
	// bit. Subtract this value by 1 to determine the length of the bitlist.
	numOfBits := uint64(8*(byteLen-1) + msb - 1)
	if numOfBits > bitLimit {
		return fmt.Errorf("too many bits")
	}
	return nil
}
