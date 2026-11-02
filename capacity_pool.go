package retry

import (
	"fmt"
	"sync"
)

// capacityPool is a thread-safe token pool. A negative maxCapacity means unlimited.
type capacityPool struct {
	available   int64
	maxCapacity int64
	mu          sync.Mutex
}

func newCapacityPool(maxCapacity int64) *capacityPool {
	return &capacityPool{
		available:   maxCapacity,
		maxCapacity: maxCapacity,
	}
}

func (p *capacityPool) acquire(cost int64) bool {
	if cost < 0 {
		panic(fmt.Sprintf("cost must not be negative, got %d", cost))
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.maxCapacity < 0 {
		return true
	}

	if p.available < cost {
		return false
	}

	p.available -= cost
	return true
}

func (p *capacityPool) release(cost int64) {
	if cost < 0 {
		panic(fmt.Sprintf("cost must not be negative, got %d", cost))
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.maxCapacity < 0 {
		return
	}

	p.available += cost
	if p.available > p.maxCapacity {
		p.available = p.maxCapacity
	}
}
