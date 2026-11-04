package retry

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

var errBoom = errors.New("boom")

func alwaysFail(calls *int) RetryableFunc {
	return func() error {
		*calls++
		return errBoom
	}
}

func TestNewFixedTokenBucketThrottleValidationAndDefaults(t *testing.T) {
	t.Run("zero values get defaults", func(t *testing.T) {
		m := NewFixedTokenBucketThrottle(FixedTokenBucketThrottleConfig{})
		assert.Equal(t, int64(defaultMaxCapacity), m.pool.maxCapacity)
		assert.Equal(t, int64(defaultMaxCapacity), m.pool.available)
		assert.Equal(t, int64(defaultThrottleCost), m.config.ThrottleCost)
		assert.Equal(t, int64(defaultThrottleCost), m.config.SuccessReleaseCost)
		assert.Equal(t, int64(defaultFirstSuccessReleaseCost), m.config.FirstSuccessReleaseCost)
	})

	t.Run("SuccessReleaseCost defaults to ThrottleCost", func(t *testing.T) {
		m := NewFixedTokenBucketThrottle(FixedTokenBucketThrottleConfig{ThrottleCost: 7})
		assert.Equal(t, int64(7), m.config.SuccessReleaseCost)
	})

	t.Run("explicit values are kept", func(t *testing.T) {
		cfg := FixedTokenBucketThrottleConfig{
			MaxCapacity:             10,
			ThrottleCost:            3,
			SuccessReleaseCost:      2,
			FirstSuccessReleaseCost: 4,
		}
		m := NewFixedTokenBucketThrottle(cfg)
		assert.Equal(t, cfg, m.config)
	})

	t.Run("negative costs get defaults", func(t *testing.T) {
		m := NewFixedTokenBucketThrottle(FixedTokenBucketThrottleConfig{
			MaxCapacity:             10,
			ThrottleCost:            -1,
			SuccessReleaseCost:      -1,
			FirstSuccessReleaseCost: -1,
		})
		assert.Equal(t, int64(defaultThrottleCost), m.config.ThrottleCost)
		assert.Equal(t, int64(defaultThrottleCost), m.config.SuccessReleaseCost)
		assert.Equal(t, int64(defaultFirstSuccessReleaseCost), m.config.FirstSuccessReleaseCost)
	})

	t.Run("negative MaxCapacity stays unlimited", func(t *testing.T) {
		m := NewFixedTokenBucketThrottle(FixedTokenBucketThrottleConfig{MaxCapacity: -5})
		assert.True(t, m.pool.unlimited())
	})
}

func TestFixedTokenBucketThrottleManagerBehavior(t *testing.T) {
	newManager := func() *FixedTokenBucketThrottleManager {
		return NewFixedTokenBucketThrottle(FixedTokenBucketThrottleConfig{
			MaxCapacity:             10,
			ThrottleCost:            5,
			SuccessReleaseCost:      3,
			FirstSuccessReleaseCost: 1,
		})
	}

	t.Run("AllowRetry consumes capacity and can deny retries", func(t *testing.T) {
		m := newManager()
		assert.True(t, m.AllowRetry(0, errBoom))
		assert.Equal(t, int64(5), m.pool.available)
		assert.True(t, m.AllowRetry(1, errBoom))
		assert.Equal(t, int64(0), m.pool.available)
		assert.False(t, m.AllowRetry(2, errBoom))
		assert.Equal(t, int64(0), m.pool.available, "available should not change on denied retry")
	})

	t.Run("OnSuccess releases FirstSuccessReleaseCost when attempt == 0", func(t *testing.T) {
		m := newManager()
		assert.True(t, m.AllowRetry(0, errBoom))
		m.OnSuccess(0)
		assert.Equal(t, int64(6), m.pool.available)
	})

	t.Run("OnSuccess releases SuccessReleaseCost when attempt > 0 and caps at max", func(t *testing.T) {
		m := newManager()
		assert.True(t, m.AllowRetry(0, errBoom))
		assert.True(t, m.AllowRetry(1, errBoom))
		m.OnSuccess(2)
		assert.Equal(t, int64(3), m.pool.available)
		m.OnSuccess(1)
		m.OnSuccess(1)
		m.OnSuccess(1)
		assert.Equal(t, int64(10), m.pool.available, "release should cap at max capacity")
	})

	t.Run("unlimited capacity never denies", func(t *testing.T) {
		m := NewFixedTokenBucketThrottle(FixedTokenBucketThrottleConfig{MaxCapacity: -1})
		for i := uint(0); i < 100; i++ {
			assert.True(t, m.AllowRetry(i, errBoom))
			m.OnSuccess(i)
		}
		assert.Equal(t, int64(-1), m.pool.available)
	})
}

func TestWithThrottleOption(t *testing.T) {
	newRetrier := func(m ThrottleManager, opts ...Option) *Retrier {
		return New(append([]Option{Attempts(10), Delay(0), MaxJitter(0), WithThrottle(m)}, opts...)...)
	}
	newManager := func() *FixedTokenBucketThrottleManager {
		return NewFixedTokenBucketThrottle(FixedTokenBucketThrottleConfig{
			MaxCapacity:             5,
			ThrottleCost:            5,
			FirstSuccessReleaseCost: 1,
		})
	}

	t.Run("throttle gates retries", func(t *testing.T) {
		m := newManager()
		calls := 0
		err := newRetrier(m).Do(alwaysFail(&calls))
		assert.Error(t, err)
		assert.Equal(t, 2, calls, "expected throttled stop after capacity exhausted")
		assert.Equal(t, int64(0), m.pool.available)
	})

	t.Run("throttle is replenished on success across runs", func(t *testing.T) {
		m := newManager()
		retrier := newRetrier(m)

		calls := 0
		err := retrier.Do(func() error {
			calls++
			if calls == 1 {
				return errBoom
			}
			return nil
		})
		assert.NoError(t, err)
		assert.Equal(t, 2, calls)
		assert.Equal(t, int64(5), m.pool.available, "expected SuccessReleaseCost after a retried success")

		calls = 0
		assert.Error(t, retrier.Do(alwaysFail(&calls)))
		assert.Equal(t, 2, calls)
		assert.Equal(t, int64(0), m.pool.available)
	})

	t.Run("first attempt success releases FirstSuccessReleaseCost", func(t *testing.T) {
		m := newManager()
		retrier := newRetrier(m)

		calls := 0
		assert.Error(t, retrier.Do(alwaysFail(&calls)))
		assert.Equal(t, int64(0), m.pool.available)

		assert.NoError(t, retrier.Do(func() error { return nil }))
		assert.Equal(t, int64(1), m.pool.available)

		calls = 0
		assert.Error(t, retrier.Do(alwaysFail(&calls)))
		assert.Equal(t, 1, calls, "expected immediate fail-fast with only partial refill")
	})

	t.Run("RetryIf passed after WithThrottle keeps throttling", func(t *testing.T) {
		m := newManager()
		calls := 0
		err := newRetrier(m, RetryIf(func(error) bool { return true })).Do(alwaysFail(&calls))
		assert.Error(t, err)
		assert.Equal(t, 2, calls)
	})

	t.Run("RetryIf false does not consume capacity", func(t *testing.T) {
		m := newManager()
		calls := 0
		err := newRetrier(m, RetryIf(func(error) bool { return false })).Do(alwaysFail(&calls))
		assert.Error(t, err)
		assert.Equal(t, 1, calls)
		assert.Equal(t, int64(5), m.pool.available)
	})

	t.Run("last attempt does not consume capacity", func(t *testing.T) {
		m := NewFixedTokenBucketThrottle(FixedTokenBucketThrottleConfig{MaxCapacity: 10, ThrottleCost: 5})
		calls := 0
		err := newRetrier(m, Attempts(2)).Do(alwaysFail(&calls))
		assert.Error(t, err)
		assert.Equal(t, 2, calls)
		assert.Equal(t, int64(5), m.pool.available, "only the one real retry should be charged")
	})

	t.Run("AttemptsForError limit does not consume capacity", func(t *testing.T) {
		m := NewFixedTokenBucketThrottle(FixedTokenBucketThrottleConfig{MaxCapacity: 10, ThrottleCost: 5})
		calls := 0
		err := newRetrier(m, AttemptsForError(2, errBoom)).Do(alwaysFail(&calls))
		assert.Error(t, err)
		assert.Equal(t, 2, calls)
		assert.Equal(t, int64(5), m.pool.available, "only the one real retry should be charged")
	})

	t.Run("denied retry skips OnRetry", func(t *testing.T) {
		m := newManager()
		var retried []uint
		calls := 0
		err := newRetrier(m, OnRetry(func(n uint, _ error) { retried = append(retried, n) })).Do(alwaysFail(&calls))
		assert.Error(t, err)
		assert.Equal(t, 2, calls)
		assert.Equal(t, []uint{0}, retried)
	})

	t.Run("throttled stop is detectable", func(t *testing.T) {
		for _, c := range []struct {
			name string
			opts []Option
		}{
			{"finite", nil},
			{"finite with LastErrorOnly", []Option{LastErrorOnly(true)}},
			{"infinite", []Option{Attempts(0)}},
		} {
			t.Run(c.name, func(t *testing.T) {
				calls := 0
				err := newRetrier(newManager(), c.opts...).Do(alwaysFail(&calls))
				assert.ErrorIs(t, err, ErrThrottled)
				assert.ErrorIs(t, err, errBoom)
				assert.Equal(t, 2, calls)
			})
		}
	})

	t.Run("exhausted attempts are not reported as throttled", func(t *testing.T) {
		for _, c := range []struct {
			name string
			opts []Option
		}{
			{"Attempts", []Option{Attempts(2)}},
			{"Attempts with LastErrorOnly", []Option{Attempts(2), LastErrorOnly(true)}},
			{"AttemptsForError", []Option{AttemptsForError(2, errBoom)}},
			{"infinite with AttemptsForError", []Option{Attempts(0), AttemptsForError(2, errBoom)}},
		} {
			t.Run(c.name, func(t *testing.T) {
				calls := 0
				err := newRetrier(newManager(), c.opts...).Do(alwaysFail(&calls))
				assert.ErrorIs(t, err, errBoom)
				assert.NotErrorIs(t, err, ErrThrottled)
				assert.Equal(t, 2, calls)
			})
		}
	})

	t.Run("infinite attempts with AttemptsForError limit do not consume capacity", func(t *testing.T) {
		m := NewFixedTokenBucketThrottle(FixedTokenBucketThrottleConfig{MaxCapacity: 10, ThrottleCost: 5})
		calls := 0
		err := newRetrier(m, Attempts(0), AttemptsForError(2, errBoom)).Do(alwaysFail(&calls))
		assert.ErrorIs(t, err, errBoom)
		assert.Equal(t, 2, calls)
		assert.Equal(t, int64(5), m.pool.available, "only the one real retry should be charged")
	})

	t.Run("infinite attempts release on success", func(t *testing.T) {
		m := newManager()
		calls := 0
		err := newRetrier(m, Attempts(0)).Do(func() error {
			calls++
			if calls == 1 {
				return errBoom
			}
			return nil
		})
		assert.NoError(t, err)
		assert.Equal(t, int64(5), m.pool.available)
	})

	t.Run("nil manager disables throttling", func(t *testing.T) {
		calls := 0
		err := newRetrier(nil).Do(alwaysFail(&calls))
		assert.Error(t, err)
		assert.Equal(t, 10, calls)
	})

	t.Run("works with RetrierWithData", func(t *testing.T) {
		m := newManager()
		calls := 0
		_, err := NewWithData[int](Attempts(10), Delay(0), MaxJitter(0), WithThrottle(m)).Do(func() (int, error) {
			calls++
			return 0, errBoom
		})
		assert.Error(t, err)
		assert.Equal(t, 2, calls)
	})
}

func TestWithFixedTokenBucketThrottleOption(t *testing.T) {
	config := FixedTokenBucketThrottleConfig{
		MaxCapacity:             5,
		ThrottleCost:            5,
		FirstSuccessReleaseCost: 1,
	}
	newRetrier := func() *Retrier {
		return New(Attempts(10), Delay(0), MaxJitter(0), WithFixedTokenBucketThrottle(config))
	}

	t.Run("bucket is shared across Do calls of one Retrier", func(t *testing.T) {
		retrier := newRetrier()
		calls := 0
		assert.Error(t, retrier.Do(alwaysFail(&calls)))
		assert.Equal(t, 2, calls)

		calls = 0
		assert.Error(t, retrier.Do(alwaysFail(&calls)))
		assert.Equal(t, 1, calls, "expected fail-fast on exhausted bucket")
	})

	t.Run("each Retrier gets its own bucket", func(t *testing.T) {
		opt := WithFixedTokenBucketThrottle(config)
		calls := 0
		assert.Error(t, New(Attempts(10), Delay(0), MaxJitter(0), opt).Do(alwaysFail(&calls)))
		assert.Equal(t, 2, calls)

		calls = 0
		assert.Error(t, New(Attempts(10), Delay(0), MaxJitter(0), opt).Do(alwaysFail(&calls)))
		assert.Equal(t, 2, calls)
	})

}
