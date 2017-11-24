package retry

import (
	"time"
)

type config struct {
	tries           uint
	delay           time.Duration
	units           time.Duration
	onRetryFunction OnRetry
	retryIfFunction RetryIfFunc
}

// Option represents an option for retry.
type Option func(*config)

// Tries set count of retry
// default is 10
func Tries(tries uint) Option {
	return func(c *config) {
		c.tries = tries
	}
}

// Delay set delay between retry
// default are 1e5 units
func Delay(delay time.Duration) Option {
	return func(c *config) {
		c.delay = delay
	}
}

// Units set unit of delay (probably only for tests purpose)
// default are microsecond
func Units(units time.Duration) Option {
	return func(c *config) {
		c.units = units
	}
}

// OnRetryFunction function callback are called each retry
//
// log each retry example:
//
//	retry.Do(
//		func() error {
//			return errors.New("some error")
//		},
//		retry.OnRetryFunction(func(n unit, err error) {
//			log.Printf("#%d: %s\n", n, err)
//		}),
//	)
func OnRetryFunction(onRetryFunction OnRetry) Option {
	return func(c *config) {
		c.onRetryFunction = onRetryFunction
	}
}

// RetryIfFunction controls whether a retry should be attempted after an error
// (assuming there are any retry attempts remaining)
//
// skip retry if special error example:
//
//	retry.Do(
//		func() error {
//			return errors.New("special error")
//		},
//		retry.RetryIfFunction(func(err error) bool {
//			if strings.Contains(err.Error, "special error") {
//				return false
//			}
//			return true
//		})
//	)
func RetryIfFunction(retryIfFunction RetryIfFunc) Option {
	return func(c *config) {
		c.retryIfFunction = retryIfFunction
	}
}
