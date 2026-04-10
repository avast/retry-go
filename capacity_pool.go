package retry

import (
	"fmt"
	"sync"
)

type CapacityPool struct {
	available   int64
	maxCapacity int64
	mu          sync.Mutex
}

func NewCapacityPool(maxCapacity int64) *CapacityPool {
	// allow negative maxCapacity to mean unlimited capacity
	return &CapacityPool{
		available:   maxCapacity,
		maxCapacity: maxCapacity,
		mu:          sync.Mutex{},
	}
}

func (p *CapacityPool) Acquire(cost int64) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if cost < 0 {
		panic(fmt.Sprintf("cost must be positive, got %d", cost))
	}

	if p.maxCapacity < 0 {
		return true
	}

	if p.available < cost {
		return false
	}

	p.available -= cost
	return true
}

func (p *CapacityPool) Release(cost int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if cost < 0 {
		panic(fmt.Sprintf("cost must be positive, got %d", cost))
	}

	if p.maxCapacity < 0 {
		return
	}

	p.available += cost
	if p.available > p.maxCapacity {
		p.available = p.maxCapacity
	}
}
