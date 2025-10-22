package bufpool

import (
	"testing"
	"unsafe"
)

func TestBufferPoolHitRate(t *testing.T) {
	pool := NewBufferPool()

	// Test sequential get/put operations with same size
	size := 128
	hits := 0
	misses := 0

	// First allocation will always be a miss (creates new buffer)
	buf1 := pool.Get(size)
	firstPtr := uintptr(unsafe.Pointer(&buf1.B[0]))
	t.Logf("First get: ptr=%p, capacity=%d", &buf1.B[0], cap(buf1.B))

	pool.Put(buf1)

	// Subsequent gets should reuse the same buffer
	for i := range 10 {
		buf := pool.Get(size)
		ptr := uintptr(unsafe.Pointer(&buf.B[0]))

		if ptr == firstPtr {
			hits++
			t.Logf("Get %d: HIT - same pointer", i+2)
		} else {
			misses++
			t.Logf("Get %d: MISS - different pointer", i+2)
		}
		pool.Put(buf)
	}

	// Don't count the first allocation as a miss for hit rate
	hitRate := float64(hits) / float64(hits+misses) * 100
	t.Logf("Hit rate: %.1f%% (%d hits, %d misses)", hitRate, hits, misses)

	if hitRate < 100 {
		t.Errorf("Expected 100%% hit rate for sequential usage, got %.1f%%", hitRate)
	}
}

func TestBufferPoolIndexCalculation(t *testing.T) {
	pool := NewBufferPool()

	// Test various sizes and their power-of-2 alignment
	testSizes := []int{1, 32, 64, 65, 128, 129, 256}

	for _, size := range testSizes {
		buf := pool.Get(size)
		t.Logf("Size %d -> capacity %d (index would be %d)",
			size, cap(buf.B), nextPowerOf2Index(size))

		// Put it back
		pool.Put(buf)

		// Get again and check if we get the same capacity
		buf2 := pool.Get(size)
		t.Logf("  Second get: capacity %d", cap(buf2.B))

		if cap(buf.B) != cap(buf2.B) {
			t.Errorf("Capacity mismatch for size %d: first=%d, second=%d",
				size, cap(buf.B), cap(buf2.B))
		}
		pool.Put(buf2)
	}
}

func TestBufferPoolCapacityMismatch(t *testing.T) {
	// Test if the issue is in capacity calculation
	pool := NewBufferPool()

	size := 64
	buf1 := pool.Get(size)
	originalCap := cap(buf1.B)

	t.Logf("Original buffer: len=%d, cap=%d", len(buf1.B), cap(buf1.B))

	// Simulate what happens in Put
	fullBuf := buf1.B[:originalCap]
	t.Logf("Full buffer before put: len=%d, cap=%d", len(fullBuf), cap(fullBuf))

	pool.Put(buf1)

	// Get again
	buf2 := pool.Get(size)
	t.Logf("Retrieved buffer: len=%d, cap=%d", len(buf2.B), cap(buf2.B))

	if cap(buf1.B) != cap(buf2.B) {
		t.Errorf("Capacity changed: original=%d, retrieved=%d", cap(buf1.B), cap(buf2.B))
	}
}

// Helper function to test nextPowerOf2Index - removed since it's already in pool.go

