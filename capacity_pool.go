package retry

import "sync"

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

func (p *capacityPool) unlimited() bool {
	return p.maxCapacity < 0
}

func (p *capacityPool) acquire(cost int64) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.unlimited() {
		return true
	}

	if p.available < cost {
		return false
	}

	p.available -= cost
	return true
}

func (p *capacityPool) release(cost int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.unlimited() {
		return
	}

	if cost >= p.maxCapacity-p.available {
		p.available = p.maxCapacity
		return
	}

	p.available += cost
}
