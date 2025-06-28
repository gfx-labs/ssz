package flexssz

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
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

type peeker struct {
	*Decoder
}

func (p *peeker) Read(o []byte) (int, error) {
	return p.Decoder.Peek(o)
}
func (d *Decoder) PeekUint32() (i uint32, err error) {
	err = binary.Read(&peeker{d}, order, &i)
	return
}
func (d *Decoder) PeekBinary(a any) (err error) {
	err = binary.Read(&peeker{d}, order, a)
	return
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

type ContainerDecoder struct {
	*Decoder
	xs [][2]uint32
}

type DecodeFunc func(*Decoder) error

type OpCode int

const (
	OpDynamicList = -2
	OpFixedList   = -1
	OpFixed       = 0
)

type Op struct {
	O OpCode
	F DecodeFunc
	M int
	S int
}

func Fixed(f DecodeFunc) Op {
	return Op{
		O: OpFixed,
		F: f,
		M: 1,
	}
}
func FixedList(f DecodeFunc, chunkSize int, maxChunks int) Op {
	return Op{
		O: OpFixedList,
		F: f,
		M: maxChunks,
		S: chunkSize,
	}
}

func DynamicList(f DecodeFunc, max int) Op {
	return Op{
		O: OpDynamicList,
		F: f,
		M: max,
	}
}

func (sd *Decoder) DecodeDynamicList(v Op) error {
	// we can calculate the number of dynamic elements in this list via the first pointer
	// this is because since offset[0] = 4 * count <==> offset[0]/4 == count
	offset, err := sd.PeekUint32()
	if err != nil {
		return err
	}
	count := int(offset / 4)
	// check if the count is too large.
	if count > v.M {
		return fmt.Errorf("ssz: dlist big (%d > %d)", len(sd.Remaining()), v.M*v.S)
	}
	// now we can read all the sub offsets
	subOffsets := []int{}
	for range count {
		oset, err := sd.ReadOffset()
		if err != nil {
			return err
		}
		subOffsets = append(subOffsets, oset)
	}
	ojdx := len(subOffsets)
	for j, vi := range subOffsets {
		if vi > len(sd.xs) {
			return fmt.Errorf("ssz: %w (%d > %d)", io.ErrUnexpectedEOF, vi, len(sd.xs))
		}
		var ssd *Decoder
		if j+1 == ojdx {
			ssd = NewDecoder(sd.xs[vi:])
		} else {
			stop := subOffsets[j+1]
			if stop > len(sd.xs) {
				return fmt.Errorf("ssz: %w (%d > %d)", io.ErrUnexpectedEOF, stop, len(sd.xs))
			}
			ssd = NewDecoder(sd.xs[vi:stop])
		}
		err = v.F(ssd)
		if err != nil {
			return err
		}
	}
	return nil
}

func (sd *Decoder) DecodeFixedList(v Op) error {
	// here we are checking the max bytes that the user input
	// this is a not bad catch-all for malformed inputs, but it won't catch everything
	if v.S*v.M > 0 && len(sd.Remaining()) > v.S*v.M {
		return fmt.Errorf("ssz: flist big (%d > %d)", len(sd.Remaining()), v.M*v.S)
	}
	// amount of chunks to read
	count := len(sd.Remaining()) / v.S
	if count > v.M {
		return fmt.Errorf("ssz: flist big (%d > %d)", count, v.M)
	}
	for range count {
		err := v.F(sd)
		if err != nil {
			return err
		}
		if _, err := sd.Read(nil); err == io.EOF {
			return nil
		}
	}
	if _, err := sd.Read(nil); err != io.EOF {
		if err != nil {
			return err
		}
		return fmt.Errorf("ssz: extra bytes (%d)", len(sd.xs)-sd.cur)
	}
	return nil
}

type subOp struct {
	op Op
	o  uint32
	f  DecodeFunc
}

func (p *Decoder) DecodeContainer(ix ...Op) error {
	// to properly parse a container, we need to know if each stack element is a pointer or not
	// this will tell us the end offsets so that we can properly decode variable size dynamic elements
	offsets := make([]subOp, 0, len(ix))
	for _, v := range ix {
		switch v.O {
		case OpDynamicList, OpFixedList: // dynamic elements
			// first read the offset, advancing the parent
			offset, err := p.ReadOffset()
			if err != nil {
				return err
			}
			// add it to the list of offset instructions
			offsets = append(offsets, subOp{
				op: v,
				o:  uint32(offset),
				f:  v.F,
			})
		case OpFixed: // not dynamic elements
			// the element isnt dynamic, so read the element like normal, advancing the parent
			err := v.F(p)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("ssz: opcode %d", v.O)
		}
	}
	oidx := len(offsets)
	for i, v := range offsets {
		var sd *Decoder
		if int(v.o) > len(p.xs) {
			return fmt.Errorf("ssz: %w (%d > %d)", io.ErrUnexpectedEOF, v.o, len(p.xs))
		}
		if i+1 < oidx {
			// find the next dynamic offset in container
			stop := int(offsets[i+1].o)
			if stop > len(p.xs) {
				return fmt.Errorf("ssz: %w (%d > %d)", io.ErrUnexpectedEOF, v.o, len(p.xs))
			}
			sd = NewDecoder(p.xs[v.o:stop])
		} else {
			// if there is not one, then we assume that it goes to the end of stream (last element)
			sd = NewDecoder(p.xs[v.o:])
		}
		switch v.op.O {
		case OpDynamicList:
			err := sd.DecodeDynamicList(v.op)
			if err != nil {
				return err
			}
		case OpFixedList:
			err := sd.DecodeFixedList(v.op)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("ssz: unknown opcode %d", v.op.O)
		}
	}
	return nil
}

func (d *Decoder) Jump() (*Decoder, int, error) {
	offset, err := d.ReadOffset()
	if err != nil {
		return nil, 0, err
	}
	if len(d.xs) < offset {
		return nil, 0, errors.New("ssz: dynamic overflow")
	}
	return NewDecoder(d.xs[offset:]), offset, nil
}

func (d *Decoder) JumpLen() (*Decoder, int, error) {
	dec, l, err := d.Jump()
	return dec, l / 4, err
}

func (d *Decoder) ReadString() (string, error) {
	bts, err := d.ReadBytes()
	if err != nil {
		return "", err
	}
	s := new(strings.Builder)
	s.Write(bts)
	return s.String(), nil
}

func (d *Decoder) ReadBytes() ([]byte, error) {
	rem := d.Remaining()
	ln := len(rem)
	d.cur = d.cur + ln
	return rem, nil
}

func (d *Decoder) ReadUint128() (*uint256.Int, error) {
	bytes, err := d.ReadN(16)
	if err != nil {
		return nil, err
	}
	
	// SSZ uses little-endian, but uint256.SetBytes expects big-endian
	// So we need to reverse the bytes
	reversed := make([]byte, 16)
	for i := 0; i < 16; i++ {
		reversed[i] = bytes[15-i]
	}
	
	val := new(uint256.Int)
	val.SetBytes(reversed)
	return val, nil
}

func (d *Decoder) ReadUint256() (*uint256.Int, error) {
	bytes, err := d.ReadN(32)
	if err != nil {
		return nil, err
	}
	
	// SSZ uses little-endian, but uint256.SetBytes expects big-endian
	// So we need to reverse the bytes
	reversed := make([]byte, 32)
	for i := 0; i < 32; i++ {
		reversed[i] = bytes[31-i]
	}
	
	val := new(uint256.Int)
	val.SetBytes(reversed)
	return val, nil
}
