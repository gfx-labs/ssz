// Package bufpool provides a thread-safe buffer pool for reusing byte slices.
// This implementation is inspired by https://github.com/libp2p/go-buffer-pool
package bufpool

import (
	"math/bits"
	"sync"
)

type Buf struct {
	B []byte
}

// BufferPool is a thread-safe pool of byte slices.
// It maintains separate pools for different buffer sizes to optimize memory usage.
type BufferPool struct {
	pools [32]sync.Pool
}

// NewBufferPool creates a new buffer pool.
func NewBufferPool() *BufferPool {
	bp := &BufferPool{}
	// Initialize pools with New functions
	for i := range bp.pools {
		size := 1 << i // power of 2
		bp.pools[i].New = func(sz int) func() any {
			return func() any {
				return &Buf{B: make([]byte, sz)}
			}
		}(size)
	}
	return bp
}

// Get returns a byte slice with at least the requested capacity.
// The returned slice may be larger than requested to align with pool sizes.
func (p *BufferPool) Get(size int) *Buf {
	if size < 0 {
		return nil
	}

	if size == 0 {
		return &Buf{B: make([]byte, 0)}
	}

	// Find the appropriate pool index (next power of 2)
	index := nextPowerOf2Index(size)
	if index >= len(p.pools) {
		// Size too large for pooling, allocate directly
		return &Buf{B: make([]byte, size)}
	}

	// Try to get a buffer from the pool
	if buf := p.pools[index].Get(); buf != nil {
		slice := buf.(*Buf)
		// Return slice with the requested length but keep the capacity
		slice.B = slice.B[:size]
		return slice
	}

	// No buffer available, create a new one
	capacity := 1 << index
	return &Buf{B: make([]byte, capacity)}
}

// Put returns a byte slice to the pool for reuse.
// The slice should not be used after calling Put.
func (p *BufferPool) Put(buf *Buf) {
	if buf == nil {
		return
	}

	capacity := cap(buf.B)
	if capacity == 0 {
		return
	}

	// Find the pool index for this capacity
	index := powerOf2Index(capacity)
	if index < 0 || index >= len(p.pools) {
		// Capacity doesn't match a pool size, don't pool it
		return
	}

	// Verify this is actually a power of 2 capacity
	if capacity != (1 << index) {
		// Not a power of 2, don't pool it
		return
	}

	// Reset the slice length to full capacity before pooling
	buf.B = buf.B[:capacity]

	// Clear the buffer before returning to pool
	for i := range buf.B {
		buf.B[i] = 0
	}

	p.pools[index].Put(buf)
}

// nextPowerOf2Index returns the index of the smallest power of 2 >= n.
// Returns the index such that 1 << index >= n.
func nextPowerOf2Index(n int) int {
	if n <= 1 {
		return 0
	}
	// Use bits.Len to find the position of the highest set bit
	// then check if n is already a power of 2
	bitLen := bits.Len(uint(n - 1))
	return bitLen
}

// powerOf2Index returns the index if n is exactly a power of 2, otherwise -1.
func powerOf2Index(n int) int {
	if n <= 0 || (n&(n-1)) != 0 {
		// Not a power of 2
		return -1
	}
	return bits.TrailingZeros(uint(n))
}

// GlobalPool is a global buffer pool instance for convenient usage.
var globalPool = NewBufferPool()

// Get is a convenience function that uses the global buffer pool.
func Get(size int) *Buf {
	return globalPool.Get(size)
}

// Put is a convenience function that uses the global buffer pool.
func Put(buf *Buf) {
	globalPool.Put(buf)
}
