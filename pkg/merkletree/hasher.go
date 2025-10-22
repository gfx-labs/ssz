package merkle_tree

import (
	"crypto/sha256"
)

// General purpose Sha256
func Sha256(data []byte, extras ...[]byte) (b [32]byte) {
	h := sha256.New()
	h.Reset()

	h.Write(data)
	for _, extra := range extras {
		h.Write(extra)
	}
	h.Sum(b[:0])
	return b
}
