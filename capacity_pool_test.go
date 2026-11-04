package retry

import (
	"math"
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCapacityPoolAcquireRelease(t *testing.T) {
	t.Run("limited capacity", func(t *testing.T) {
		p := newCapacityPool(10)

		ok := p.acquire(5)
		assert.True(t, ok)
		assert.Equal(t, int64(5), p.available)

		ok = p.acquire(6)
		assert.False(t, ok)
		assert.Equal(t, int64(5), p.available, "available should not change on failed acquire")

		p.release(5)
		assert.Equal(t, int64(10), p.available)

		ok = p.acquire(6)
		assert.True(t, ok)
		assert.Equal(t, int64(4), p.available)
	})

	t.Run("release is capped by maxCapacity", func(t *testing.T) {
		p := newCapacityPool(10)
		p.release(5)
		assert.Equal(t, int64(10), p.available)
	})

	t.Run("zero cost is allowed", func(t *testing.T) {
		p := newCapacityPool(10)
		assert.True(t, p.acquire(0))
		assert.Equal(t, int64(10), p.available)
		p.release(0)
		assert.Equal(t, int64(10), p.available)
	})

	t.Run("unlimited capacity", func(t *testing.T) {
		p := newCapacityPool(-1)
		assert.Equal(t, int64(-1), p.maxCapacity)

		ok := p.acquire(1_000_000)
		assert.True(t, ok)
		assert.Equal(t, int64(-1), p.available, "available should remain unchanged for unlimited capacity")

		p.release(1_000_000)
		assert.Equal(t, int64(-1), p.available, "release should be a no-op for unlimited capacity")
	})
}

func TestCapacityPoolReleaseDoesNotOverflow(t *testing.T) {
	for _, c := range []struct {
		name     string
		acquire  int64
		release  int64
		expected int64
	}{
		{"release far above headroom", 1, math.MaxInt64, math.MaxInt64},
		{"release exactly the headroom", 10, 10, math.MaxInt64},
		{"release below the headroom", 10, 9, math.MaxInt64 - 1},
	} {
		t.Run(c.name, func(t *testing.T) {
			p := newCapacityPool(math.MaxInt64)
			assert.True(t, p.acquire(c.acquire))
			p.release(c.release)
			assert.Equal(t, c.expected, p.available)
		})
	}
}

func TestCapacityPoolConcurrency(t *testing.T) {
	t.Run("limited capacity invariants hold", func(t *testing.T) {
		const max int64 = 3
		p := newCapacityPool(max)

		const goroutines = 32
		const iterations = 2_000

		var wg sync.WaitGroup
		wg.Add(goroutines)
		for g := 0; g < goroutines; g++ {
			go func() {
				defer wg.Done()
				for i := 0; i < iterations; i++ {
					if !p.acquire(1) {
						runtime.Gosched()
						continue
					}

					p.mu.Lock()
					avail := p.available
					p.mu.Unlock()
					assert.GreaterOrEqual(t, avail, int64(0), "available should never be negative")
					assert.LessOrEqual(t, avail, p.maxCapacity, "available should never exceed maxCapacity")

					p.release(1)

					p.mu.Lock()
					avail = p.available
					p.mu.Unlock()
					assert.GreaterOrEqual(t, avail, int64(0), "available should never be negative")
					assert.LessOrEqual(t, avail, p.maxCapacity, "available should never exceed maxCapacity")
				}
			}()
		}
		wg.Wait()

		p.mu.Lock()
		finalAvail := p.available
		p.mu.Unlock()
		assert.GreaterOrEqual(t, finalAvail, int64(0))
		assert.LessOrEqual(t, finalAvail, p.maxCapacity)
	})

	t.Run("unlimited capacity stays unlimited", func(t *testing.T) {
		p := newCapacityPool(-1)

		const goroutines = 32
		const iterations = 2_000

		var wg sync.WaitGroup
		wg.Add(goroutines)
		for g := 0; g < goroutines; g++ {
			go func() {
				defer wg.Done()
				for i := 0; i < iterations; i++ {
					assert.True(t, p.acquire(1))
					p.release(1)
				}
			}()
		}
		wg.Wait()

		p.mu.Lock()
		defer p.mu.Unlock()
		assert.Equal(t, int64(-1), p.maxCapacity)
		assert.Equal(t, int64(-1), p.available)
	})
}
