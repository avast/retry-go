package retry

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewFixTokenBucketThrottleValidationAndDefaults(t *testing.T) {
	t.Run("panics on negative throttle cost", func(t *testing.T) {
		assert.PanicsWithValue(t, "ThrottleCost must be positive, got -1", func() {
			_ = NewFixTokenBucketThrottle(FixTokenBucketThrottleConfig{
				MaxCapacity:  10,
				ThrottleCost: -1,
			})
		})
	})

	t.Run("panics on negative first success release", func(t *testing.T) {
		assert.PanicsWithValue(t, "FirstSuccessRelease must be positive, got -1", func() {
			_ = NewFixTokenBucketThrottle(FixTokenBucketThrottleConfig{
				MaxCapacity:         10,
				ThrottleCost:        5,
				FirstSuccessRelease: -1,
			})
		})
	})

	t.Run("defaults are applied when zero", func(t *testing.T) {
		m := NewFixTokenBucketThrottle(FixTokenBucketThrottleConfig{
			MaxCapacity:         10,
			ThrottleCost:        0,
			FirstSuccessRelease: 0,
		})
		assert.Equal(t, int64(10), m.retryPool.maxCapacity)
		assert.Equal(t, int64(10), m.retryPool.available)
		assert.Equal(t, int64(defaultThrottleCost), m.config.ThrottleCost)
		assert.Equal(t, int64(defaultFirstSuccessRelease), m.config.FirstSuccessRelease)
	})
}

func TestFixTokenBucketThrottleManagerBehavior(t *testing.T) {
	t.Run("retryIf consumes capacity and can deny retries", func(t *testing.T) {
		m := NewFixTokenBucketThrottle(FixTokenBucketThrottleConfig{
			MaxCapacity:         10,
			ThrottleCost:        5,
			FirstSuccessRelease: 1,
		})

		assert.True(t, m.RetryIf(errors.New("boom")))
		assert.Equal(t, int64(5), m.retryPool.available)

		assert.True(t, m.RetryIf(errors.New("boom")))
		assert.Equal(t, int64(0), m.retryPool.available)

		assert.False(t, m.RetryIf(errors.New("boom")))
		assert.Equal(t, int64(0), m.retryPool.available, "available should not change on denied retry")
	})

	t.Run("onSuccess releases first-success amount when attempt == 0", func(t *testing.T) {
		m := NewFixTokenBucketThrottle(FixTokenBucketThrottleConfig{
			MaxCapacity:         10,
			ThrottleCost:        5,
			FirstSuccessRelease: 1,
		})

		assert.True(t, m.RetryIf(errors.New("boom")))
		assert.Equal(t, int64(5), m.retryPool.available)

		m.OnSuccess(0)
		assert.Equal(t, int64(6), m.retryPool.available)
	})

	t.Run("onSuccess releases throttle cost when attempt > 0 and caps at max", func(t *testing.T) {
		m := NewFixTokenBucketThrottle(FixTokenBucketThrottleConfig{
			MaxCapacity:         10,
			ThrottleCost:        5,
			FirstSuccessRelease: 1,
		})

		assert.True(t, m.RetryIf(errors.New("boom")))
		assert.True(t, m.RetryIf(errors.New("boom")))
		assert.Equal(t, int64(0), m.retryPool.available)

		m.OnSuccess(2)
		assert.Equal(t, int64(5), m.retryPool.available)

		m.OnSuccess(3)
		assert.Equal(t, int64(10), m.retryPool.available, "release should cap at max capacity")
	})

	t.Run("unlimited capacity never denies and doesn't change available", func(t *testing.T) {
		m := NewFixTokenBucketThrottle(FixTokenBucketThrottleConfig{
			MaxCapacity:         -1,
			ThrottleCost:        5,
			FirstSuccessRelease: 1,
		})
		assert.Equal(t, int64(-1), m.retryPool.available)

		for i := 0; i < 100; i++ {
			assert.True(t, m.RetryIf(errors.New("boom")))
			m.OnSuccess(uint(i))
		}
		assert.Equal(t, int64(-1), m.retryPool.available)
	})
}

func TestWithFixTokenBucketThrottleOption(t *testing.T) {
	t.Run("throttle gates retries", func(t *testing.T) {
		manager := NewFixTokenBucketThrottle(FixTokenBucketThrottleConfig{
			MaxCapacity:         5,
			ThrottleCost:        5,
			FirstSuccessRelease: 1,
		})
		retrier := New(
			Attempts(10),
			Delay(0),
			MaxJitter(0),
			WithThrottle(manager),
		)

		calls := 0
		err := retrier.Do(func() error {
			calls++
			return errors.New("always fails")
		})
		assert.Error(t, err)
		assert.Equal(t, 2, calls, "expected throttled stop after capacity exhausted")
		assert.Equal(t, int64(0), manager.retryPool.available, "throttle capacity should be exhausted after gating")
	})

	t.Run("throttle is replenished on success across runs", func(t *testing.T) {
		manager := NewFixTokenBucketThrottle(FixTokenBucketThrottleConfig{
			MaxCapacity:         5,
			ThrottleCost:        5,
			FirstSuccessRelease: 1,
		})
		retrier := New(
			Attempts(10),
			Delay(0),
			MaxJitter(0),
			WithThrottle(manager),
		)

		// First run: fail once, then succeed.
		calls := 0
		err := retrier.Do(func() error {
			calls++
			if calls == 1 {
				return errors.New("fail once")
			}
			return nil
		})
		assert.NoError(t, err)
		assert.Equal(t, 2, calls, "expected one retry before success")
		assert.Equal(t, int64(5), manager.retryPool.available, "expected ThrottleCost to be released on success after retries")

		// Second run: always fail. If the throttle wasn't replenished, we'd stop after 1 call.
		calls = 0
		err = retrier.Do(func() error {
			calls++
			return errors.New("always fails")
		})
		assert.Error(t, err)
		assert.Equal(t, 2, calls, "expected throttled stop after capacity exhausted")
		assert.Equal(t, int64(0), manager.retryPool.available, "throttle capacity should be exhausted again")
	})

	t.Run("first call success releases FirstSuccessRelease", func(t *testing.T) {
		manager := NewFixTokenBucketThrottle(FixTokenBucketThrottleConfig{
			MaxCapacity:         5,
			ThrottleCost:        5,
			FirstSuccessRelease: 1,
		})
		retrier := New(
			Attempts(10),
			Delay(0),
			MaxJitter(0),
			WithThrottle(manager),
		)

		// Exhaust capacity via the retrier flow (first retryIf acquires 5, second is denied).
		calls := 0
		err := retrier.Do(func() error {
			calls++
			return errors.New("always fails")
		})
		assert.Error(t, err)
		assert.Equal(t, 2, calls, "expected throttled stop after capacity exhausted")
		assert.Equal(t, int64(0), manager.retryPool.available)

		// Now succeed on the first attempt: attempt == 0 should release FirstSuccessRelease.
		err = retrier.Do(func() error { return nil })
		assert.NoError(t, err)
		assert.Equal(t, int64(1), manager.retryPool.available, "expected FirstSuccessRelease when attempt == 0")
	})
}
