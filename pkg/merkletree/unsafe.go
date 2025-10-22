package merkle_tree

import "unsafe"

// singleToChunked is a helper function to convert a slice of bytes to a slice of [32]byte with no copy or allocation
// it uses unsafe.
func singleToChunked(xs []byte) [][32]byte {
	if len(xs) == 0 {
		return nil
	}
	// get the ptr to the slice data
	// change the type of the ptr. think like c
	return unsafe.Slice((*[32]byte)(unsafe.Pointer(unsafe.SliceData(xs))), len(xs)>>5)
}

// chunkedToSingle is a helper function to convert a slice of [32]byte to a slice of bytes with no copy or allocation
// it uses unsafe.
func chunkedToSingle(xs [][32]byte) []byte {
	if len(xs) == 0 {
		return nil
	}
	// then we move over the values
	return unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(xs))), len(xs)<<5)
}
