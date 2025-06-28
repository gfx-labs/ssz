package ssz

import "fmt"

// SizeMismatchError represents an error when the actual size doesn't match the expected size
type SizeMismatchError struct {
	Expected int
	Got      int
}

// NewSizeMismatchError creates a new SizeMismatchError
func NewSizeMismatchError(expected, got int) *SizeMismatchError {
	return &SizeMismatchError{
		Expected: expected,
		Got:      got,
	}
}

// Error implements the error interface
func (e *SizeMismatchError) Error() string {
	return fmt.Sprintf("size mismatch: expected %d bytes, got %d bytes", e.Expected, e.Got)
}