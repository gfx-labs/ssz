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
	err := p.DecodeContainer([]Op{
		Fixed(func(d *Decoder) error {
			return d.ScanUint64(&s.v)
		}),
		FixedList(func(d *Decoder) (err error) {
			s.s, err = d.ReadString()
			return
		}, 1, 32),
		FixedList(func(d *Decoder) error {
			i, err := d.ReadUint64()
			if err != nil {
				return err
			}
			s.xs = append(s.xs, i)
			return nil
		}, 8, 32),
	}...)
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
	err := p.DecodeContainer([]Op{
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
}
