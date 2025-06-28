package flexssz

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeEncoder(t *testing.T) {
	buf := new(bytes.Buffer)
	b := NewBuilder(buf)
	b.EncodeUint64(5555).
		EncodeString("1234").
		EnterDynamic().
		EncodeUint64(41).
		EncodeUint64(42).
		EncodeUint64(43).
		EncodeUint64(44).
		EncodeUint64(45).
		EncodeUint64(46).
		EncodeUint64(47).
		EncodeUint64(48).
		ExitDynamic().Finish()

	type S struct {
		v  uint64
		s  string
		xs []uint64
	}
	var s S
	ans := buf.Bytes()
	p := NewDecoder(ans)
	
	// Decode using DecodeContainer
	err := p.DecodeContainer(
		Fixed(func(d *Decoder) error {
			return d.ScanUint64(&s.v)
		}),
		Variable(func(d *Decoder) error {
			buf, err := d.ReadAll()
			if err == nil {
				s.s = string(buf)
			}
			return err
		}),
		Variable(func(d *Decoder) error {
			for d.cur < len(d.xs) {
				i, err := d.ReadUint64()
				if err != nil {
					break
				}
				s.xs = append(s.xs, i)
			}
			return nil
		}),
	)
	require.NoError(t, err)
}

func TestDecodeEncoder2(t *testing.T) {
	buf := new(bytes.Buffer)
	b := NewBuilder(buf)
	b.EncodeUint64(5555).
		EnterDynamic().
		EnterDynamic().
		EncodeUint64(41).
		EncodeUint64(42).
		EncodeUint64(43).
		ExitDynamic().
		EnterDynamic().
		ExitDynamic().
		EnterDynamic().
		ExitDynamic().
		EnterDynamic().
		EncodeUint64(41).
		EncodeUint64(42).
		EncodeUint64(43).
		EncodeUint64(44).
		EncodeUint64(45).
		EncodeUint64(46).
		EncodeUint64(47).
		EncodeUint64(48).
		ExitDynamic().
		ExitDynamic().
		Finish()

	type S struct {
		v  uint64
		xs [][]uint64
	}
	var s S
	ans := buf.Bytes()
	p := NewDecoder(ans)
	
	// Decode using DecodeContainer
	err := p.DecodeContainer(
		Fixed(func(d *Decoder) error {
			return d.ScanUint64(&s.v)
		}),
		Variable(func(d *Decoder) error {
			// Read first offset to get count
			firstOffset, err := d.PeekUint32()
			if err != nil {
				return err
			}
			count := firstOffset / 4
			
			// Read all offsets
			offsets := make([]int, count)
			for i := 0; i < int(count); i++ {
				off, err := d.ReadOffset()
				if err != nil {
					return err
				}
				offsets[i] = off
			}
			
			// Decode each sublist
			for i, off := range offsets {
				var endOff int
				if i+1 < len(offsets) {
					endOff = offsets[i+1]
				} else {
					endOff = d.Len()
				}
				
				// Create decoder for this sublist
				subDecoder := NewDecoder(d.xs[off:endOff])
				
				// Read the fixed list of uint64s
				xs := []uint64{}
				for subDecoder.cur < len(subDecoder.xs) {
					val, err := subDecoder.ReadUint64()
					if err != nil {
						break
					}
					xs = append(xs, val)
				}
				s.xs = append(s.xs, xs)
			}
			return nil
		}),
	)
	require.NoError(t, err)
}
