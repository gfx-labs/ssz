package ssz

import "fmt"

// ErrSizeMismatch represents an error when the actual size doesn't match the expected size
type ErrSizeMismatch struct {
	Expected int
	Got      int
}

// NewErrSizeMismatch creates a new ErrSizeMismatch
func NewErrSizeMismatch(expected, got int) *ErrSizeMismatch {
	return &ErrSizeMismatch{
		Expected: expected,
		Got:      got,
	}
}

// Error implements the error interface
func (e *ErrSizeMismatch) Error() string {
	return fmt.Sprintf("size mismatch: expected %d bytes, got %d bytes", e.Expected, e.Got)
}

