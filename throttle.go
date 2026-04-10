package retry

import "fmt"

const defaultMaxCapacity = 100
const defaultThrottleCost = 5
const defaultFirstSuccessRelease = 1

type ThrottleManager interface {
	RetryIf(err error) bool
	OnSuccess(attempt uint)
}

type FixTokenBucketThrottleManager struct {
	retryPool *CapacityPool
	config    FixTokenBucketThrottleConfig
}

type FixTokenBucketThrottleConfig struct {
	MaxCapacity         int64
	ThrottleCost        int64
	FirstSuccessRelease int64
}

func NewFixTokenBucketThrottle(config FixTokenBucketThrottleConfig) *FixTokenBucketThrottleManager {
	if config.MaxCapacity == 0 {
		config.MaxCapacity = defaultMaxCapacity
	}

	if config.ThrottleCost < 0 {
		panic(fmt.Sprintf("ThrottleCost must be positive, got %d", config.ThrottleCost))
	}
	if config.ThrottleCost == 0 {
		config.ThrottleCost = defaultThrottleCost
	}

	if config.FirstSuccessRelease < 0 {
		panic(fmt.Sprintf("FirstSuccessRelease must be positive, got %d", config.FirstSuccessRelease))
	}
	if config.FirstSuccessRelease == 0 {
		config.FirstSuccessRelease = defaultFirstSuccessRelease
	}

	retryPool := NewCapacityPool(config.MaxCapacity)
	return &FixTokenBucketThrottleManager{
		retryPool: retryPool,
		config:    config,
	}
}

func (m *FixTokenBucketThrottleManager) RetryIf(err error) bool {
	return m.retryPool.Acquire(m.config.ThrottleCost)
}

func (m *FixTokenBucketThrottleManager) OnSuccess(attempt uint) {
	releaseCost := m.config.ThrottleCost
	if attempt == 0 {
		releaseCost = m.config.FirstSuccessRelease
	}
	m.retryPool.Release(releaseCost)
}
