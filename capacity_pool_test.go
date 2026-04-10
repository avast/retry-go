package retry

import (
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCapacityPoolAcquireRelease(t *testing.T) {
	t.Run("limited capacity", func(t *testing.T) {
		p := NewCapacityPool(10)

		ok := p.Acquire(5)
		assert.True(t, ok)
		assert.Equal(t, int64(5), p.available)

		ok = p.Acquire(6)
		assert.False(t, ok)
		assert.Equal(t, int64(5), p.available, "available should not change on failed acquire")

		p.Release(5)
		assert.Equal(t, int64(10), p.available)

		ok = p.Acquire(6)
		assert.True(t, ok)
		assert.Equal(t, int64(4), p.available)
	})

	t.Run("release is capped by maxCapacity", func(t *testing.T) {
		p := NewCapacityPool(10)
		p.Release(5)
		assert.Equal(t, int64(10), p.available)
	})

	t.Run("zero cost is allowed", func(t *testing.T) {
		p := NewCapacityPool(10)
		assert.True(t, p.Acquire(0))
		assert.Equal(t, int64(10), p.available)
		p.Release(0)
		assert.Equal(t, int64(10), p.available)
	})

	t.Run("unlimited capacity", func(t *testing.T) {
		p := NewCapacityPool(-1)
		assert.Equal(t, int64(-1), p.maxCapacity)

		ok := p.Acquire(1_000_000)
		assert.True(t, ok)
		assert.Equal(t, int64(-1), p.available, "available should remain unchanged for unlimited capacity")

		p.Release(1_000_000)
		assert.Equal(t, int64(-1), p.available, "release should be a no-op for unlimited capacity")
	})
}

func TestCapacityPoolNegativeCostPanics(t *testing.T) {
	p := NewCapacityPool(10)

	t.Run("acquire", func(t *testing.T) {
		assert.PanicsWithValue(t, "cost must be positive, got -1", func() {
			_ = p.Acquire(-1)
		})
	})

	t.Run("release", func(t *testing.T) {
		assert.PanicsWithValue(t, "cost must be positive, got -1", func() {
			p.Release(-1)
		})
	})
}

func TestCapacityPoolConcurrency(t *testing.T) {
	t.Run("limited capacity invariants hold", func(t *testing.T) {
		const max int64 = 3
		p := NewCapacityPool(max)

		const goroutines = 32
		const iterations = 2_000

		var wg sync.WaitGroup
		wg.Add(goroutines)
		for g := 0; g < goroutines; g++ {
			go func() {
				defer wg.Done()
				for i := 0; i < iterations; i++ {
					if !p.Acquire(1) {
						runtime.Gosched()
						continue
					}

					p.mu.Lock()
					avail := p.available
					p.mu.Unlock()
					assert.GreaterOrEqual(t, avail, int64(0), "available should never be negative")
					assert.LessOrEqual(t, avail, p.maxCapacity, "available should never exceed maxCapacity")

					p.Release(1)

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
		p := NewCapacityPool(-1)

		const goroutines = 32
		const iterations = 2_000

		var wg sync.WaitGroup
		wg.Add(goroutines)
		for g := 0; g < goroutines; g++ {
			go func() {
				defer wg.Done()
				for i := 0; i < iterations; i++ {
					assert.True(t, p.Acquire(1))
					p.Release(1)
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
