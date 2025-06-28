package flexssz

import (
	"errors"
	"fmt"
)

var ErrIndexOutOfBounds = errors.New("index out of bounds")
var ErrInvalidSeek = errors.New("invalid seek offset")

type errIndexOutOfBounds struct {
	sz  int
	bad int
}

func (i *errIndexOutOfBounds) Unwrap() error {
	return ErrIndexOutOfBounds
}
func (i *errIndexOutOfBounds) Error() string {
	return fmt.Sprintf("index out of bounds: %d (%d)", i.bad, i.sz)
}

func NewErrIndexOutOfBounds(offset int, length int) error {
	return &errIndexOutOfBounds{
		sz:  length,
		bad: offset,
	}
}
