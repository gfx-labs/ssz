package solidssz

// Runtime types and utilities for solidssz generated code

// Marshaler is the interface for SSZ marshaling
type Marshaler interface {
	MarshalSSZ() ([]byte, error)
	SizeSSZ() int
}

// FixedSize indicates a type has fixed size
type FixedSize interface {
	Fixed() bool
}
