package retry

import "errors"

const (
	defaultMaxCapacity             = 100
	defaultThrottleCost            = 5
	defaultFirstSuccessReleaseCost = 1
)

// ErrThrottled is part of the error returned when a ThrottleManager denies
// a retry, alongside the attempt errors. Check it with errors.Is.
var ErrThrottled = errors.New("retry throttled")

// ThrottleManager limits retries across calls, see WithThrottle.
// Implementations must be safe for concurrent use.
type ThrottleManager interface {
	// AllowRetry is called after attempt failed with err, only when a retry
	// would otherwise follow, and before the delay. Returning false stops
	// retrying immediately with an error matching ErrThrottled.
	// If the context is cancelled during the delay, no retry happens even
	// though AllowRetry returned true.
	AllowRetry(attempt uint, err error) bool
	// OnSuccess is called when attempt succeeded (0 is the first attempt).
	OnSuccess(attempt uint)
}

// FixedTokenBucketThrottleConfig configures a FixedTokenBucketThrottleManager.
// Zero and negative costs are replaced by defaults.
type FixedTokenBucketThrottleConfig struct {
	// MaxCapacity is the bucket size: 0 means the default (100),
	// a negative value means unlimited (never throttles).
	MaxCapacity int64
	// ThrottleCost is taken from the bucket for each retry (default 5).
	ThrottleCost int64
	// SuccessReleaseCost is returned to the bucket on a success after retries
	// (default ThrottleCost).
	SuccessReleaseCost int64
	// FirstSuccessReleaseCost is returned to the bucket on a first-attempt
	// success (default 1).
	FirstSuccessReleaseCost int64
}

// FixedTokenBucketThrottleManager is a ThrottleManager backed by a
// fixed-size token bucket: retries take tokens, successes return them
// (capped at MaxCapacity). Retries are denied while the bucket lacks
// ThrottleCost tokens.
type FixedTokenBucketThrottleManager struct {
	pool   *capacityPool
	config FixedTokenBucketThrottleConfig
}

// NewFixedTokenBucketThrottle creates a FixedTokenBucketThrottleManager with
// a full bucket.
func NewFixedTokenBucketThrottle(config FixedTokenBucketThrottleConfig) *FixedTokenBucketThrottleManager {
	if config.MaxCapacity == 0 {
		config.MaxCapacity = defaultMaxCapacity
	}
	if config.ThrottleCost <= 0 {
		config.ThrottleCost = defaultThrottleCost
	}
	if config.SuccessReleaseCost <= 0 {
		config.SuccessReleaseCost = config.ThrottleCost
	}
	if config.FirstSuccessReleaseCost <= 0 {
		config.FirstSuccessReleaseCost = defaultFirstSuccessReleaseCost
	}

	return &FixedTokenBucketThrottleManager{
		pool:   newCapacityPool(config.MaxCapacity),
		config: config,
	}
}

// AllowRetry takes ThrottleCost tokens, or returns false if not enough are left.
func (m *FixedTokenBucketThrottleManager) AllowRetry(_ uint, _ error) bool {
	return m.pool.acquire(m.config.ThrottleCost)
}

// OnSuccess returns FirstSuccessReleaseCost tokens when attempt is 0,
// SuccessReleaseCost otherwise.
func (m *FixedTokenBucketThrottleManager) OnSuccess(attempt uint) {
	if attempt == 0 {
		m.pool.release(m.config.FirstSuccessReleaseCost)
		return
	}
	m.pool.release(m.config.SuccessReleaseCost)
}
