package bufpool

import (
	"runtime"
	"strconv"
	"sync"
	"testing"
)

func TestBufferPool_GetNegativeSize(t *testing.T) {
	pool := NewBufferPool()
	buf := pool.Get(-1)
	if buf != nil {
		t.Error("Get(-1) should return nil")
	}
}

func TestBufferPool_PutAndReuse(t *testing.T) {
	pool := NewBufferPool()

	// Get a buffer
	buf1 := pool.Get(64)
	if cap(buf1.B) != 64 {
		t.Fatalf("Expected capacity 64, got %d", cap(buf1.B))
	}

	// Modify the buffer
	copy(buf1.B, []byte("hello"))

	// Put it back
	pool.Put(buf1)

	// Get another buffer of the same size
	buf2 := pool.Get(64)
	if cap(buf2.B) != 64 {
		t.Fatalf("Expected capacity 64, got %d", cap(buf2.B))
	}

	// Should be cleared
	for i, b := range buf2.B {
		if b != 0 {
			t.Errorf("Buffer not cleared at index %d: got %d, want 0", i, b)
		}
	}
}

func TestBufferPool_PutNil(t *testing.T) {
	pool := NewBufferPool()
	// Should not panic
	pool.Put(nil)
}

func TestBufferPool_PutZeroCapacity(t *testing.T) {
	pool := NewBufferPool()
	buf := make([]byte, 0, 0)
	// Should not panic
	pool.Put(&Buf{buf})
}

func TestBufferPool_PutNonPowerOf2(t *testing.T) {
	pool := NewBufferPool()

	// Create a buffer with non-power-of-2 capacity
	buf := &Buf{make([]byte, 100, 100)}

	// Should not panic, but won't be pooled
	pool.Put(buf)

	// Getting a buffer of size 100 should create a new one (capacity 128)
	buf2 := pool.Get(100)
	if cap(buf2.B) != 128 {
		t.Errorf("Expected capacity 128, got %d", cap(buf2.B))
	}
}

func TestGlobalPool(t *testing.T) {
	buf := Get(32)
	if len(buf.B) != 32 {
		t.Errorf("Get(32) length = %d, want 32", len(buf.B))
	}
	if cap(buf.B) != 32 {
		t.Errorf("Get(32) capacity = %d, want 32", cap(buf.B))
	}

	Put(buf)

	// Get another buffer - should reuse
	buf2 := Get(32)
	if cap(buf2.B) != 32 {
		t.Errorf("Get(32) capacity = %d, want 32", cap(buf2.B))
	}
}

func TestNextPowerOf2Index(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{0, 0},
		{1, 0},
		{2, 1},
		{3, 2},
		{4, 2},
		{5, 3},
		{16, 4},
		{17, 5},
		{1024, 10},
		{1025, 11},
	}

	for _, tt := range tests {
		result := nextPowerOf2Index(tt.input)
		if result != tt.expected {
			t.Errorf("nextPowerOf2Index(%d) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}

func TestPowerOf2Index(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{0, -1},
		{1, 0},
		{2, 1},
		{3, -1},
		{4, 2},
		{5, -1},
		{8, 3},
		{16, 4},
		{32, 5},
		{1024, 10},
		{1025, -1},
	}

	for _, tt := range tests {
		result := powerOf2Index(tt.input)
		if result != tt.expected {
			t.Errorf("powerOf2Index(%d) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}

func TestBufferPool_Concurrent(t *testing.T) {
	pool := NewBufferPool()
	const numGoroutines = 100
	const numOperations = 1000

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := range numGoroutines {
		go func(id int) {
			defer wg.Done()

			for j := range numOperations {
				size := 64 + (id*j)%512 // Vary the size
				buf := pool.Get(size)

				// Write some data
				if len(buf.B) > 0 {
					buf.B[0] = byte(id)
				}
				if len(buf.B) > 1 {
					buf.B[len(buf.B)-1] = byte(j)
				}

				pool.Put(buf)
			}
		}(i)
	}

	wg.Wait()
}

func BenchmarkBufferPool_Get(b *testing.B) {
	pool := NewBufferPool()
	sizes := []int{64, 256, 1024, 4096}
	for _, size := range sizes {
		pool.Get(size)
	}
	for _, size := range sizes {
		b.Run(func() string { return "size_" + strconv.Itoa(size) }(), func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				buf := pool.Get(size)
				pool.Put(buf)
			}
		})
	}
}

func BenchmarkBufferPool_GetPut(b *testing.B) {
	pool := NewBufferPool()

	b.ResetTimer()
	i := 0
	for b.Loop() {
		buf := pool.Get(1024)
		// Simulate some work
		if len(buf.B) > 0 {
			buf.B[0] = byte(i)
		}
		pool.Put(buf)
		i++
	}
}

func BenchmarkDirectAllocation(b *testing.B) {
	b.ResetTimer()
	i := 0
	for b.Loop() {
		buf := make([]byte, 1024)
		// Simulate some work
		buf[0] = byte(i)
		// No explicit deallocation
		_ = buf
		i++
	}
}

func BenchmarkBufferPool_Parallel(b *testing.B) {
	pool := NewBufferPool()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := pool.Get(1024)
			if len(buf.B) > 0 {
				buf.B[0] = 42
			}
			pool.Put(buf)
		}
	})
}

func TestBufferPool_MemoryLeakProtection(t *testing.T) {
	pool := NewBufferPool()

	// Get and put many buffers
	for range 1000 {
		buf := pool.Get(1024)
		copy(buf.B, []byte("sensitive data that should be cleared"))
		pool.Put(buf)
	}

	// Force garbage collection
	runtime.GC()
	runtime.GC()

	// Get a new buffer and verify it's cleared
	buf := pool.Get(1024)
	for i, b := range buf.B {
		if b != 0 {
			t.Errorf("Buffer not cleared at index %d: got %d, want 0", i, b)
		}
	}
}

func TestBufferPool_NoAllocationOnAppend(t *testing.T) {
	pool := NewBufferPool()

	tests := []struct {
		name           string
		initialSize    int
		appendSize     int
		expectedMinCap int
	}{
		{
			name:           "small buffer with padding",
			initialSize:    64,
			appendSize:     32,
			expectedMinCap: 96,
		},
		{
			name:           "large buffer with padding",
			initialSize:    1000,
			appendSize:     24,
			expectedMinCap: 1024,
		},
		{
			name:           "exact power of 2 with small append",
			initialSize:    512,
			appendSize:     32,
			expectedMinCap: 544,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Request buffer with enough space for initial data + append
			requestSize := tt.initialSize + tt.appendSize
			buf := pool.Get(requestSize)

			// Verify we got enough capacity
			if cap(buf.B) < tt.expectedMinCap {
				t.Errorf("Buffer capacity %d is less than expected minimum %d", cap(buf.B), tt.expectedMinCap)
			}

			// Set initial length and fill with data
			buf.B = buf.B[:tt.initialSize]
			for i := range buf.B {
				buf.B[i] = byte(i % 256)
			}

			// Record original capacity
			originalCap := cap(buf.B)

			// Append data - should not cause allocation
			appendData := make([]byte, tt.appendSize)
			for i := range appendData {
				appendData[i] = 0xFF
			}
			buf.B = append(buf.B, appendData...)

			// Verify capacity didn't change (no allocation occurred)
			if cap(buf.B) != originalCap {
				t.Errorf("Buffer capacity changed from %d to %d - allocation occurred!", originalCap, cap(buf.B))
			}

			// Verify length is correct
			expectedLen := tt.initialSize + tt.appendSize
			if len(buf.B) != expectedLen {
				t.Errorf("Buffer length %d, expected %d", len(buf.B), expectedLen)
			}

			pool.Put(buf)
		})
	}
}

// simulateWorkloadWithAppends tests a realistic pattern where we get a buffer,
// fill it with data, and then append additional data without causing allocations.
func TestBufferPool_SimulateWorkloadWithAppends(t *testing.T) {
	pool := NewBufferPool()

	for i := range 100 {
		// Simulate getting a buffer for some base data plus potential appends
		baseSize := 96  // 3 chunks of 32 bytes
		extraSize := 32 // Space for one additional chunk

		buf := pool.Get(baseSize + extraSize)

		// Fill with initial data
		buf.B = buf.B[:baseSize]
		for j := range buf.B {
			buf.B[j] = byte(j + i)
		}

		// Simulate conditional append (like odd-length layer padding)
		if (len(buf.B)/32)%2 != 0 {
			originalCap := cap(buf.B)
			// Append extra data
			buf.B = append(buf.B, make([]byte, 32)...)

			// Should not have caused allocation
			if cap(buf.B) != originalCap {
				t.Fatalf("Iteration %d: allocation occurred during append", i)
			}
		}

		pool.Put(buf)
	}
}
