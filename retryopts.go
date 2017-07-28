package retry

import (
	"time"
)

// Struct for configure retry
// tries - count of tries
// delay - waiting time
// units - waiting time unit (for tests purpose)
type RetryOpts struct {
	tries uint
	delay time.Duration
	units time.Duration
}

var defaultTries uint = 10
var defaultDelay time.Duration = 1e5

// Create new RetryOpts struct with default values
// default tries are 10
// default delay are 1e5
// default units are microsecond
func NewRetryOpts() RetryOpts {
	return RetryOpts{tries: defaultTries, delay: defaultDelay, units: time.Microsecond}
}

// Units setter
func (opts RetryOpts) Units(timeUnit time.Duration) RetryOpts {
	opts.units = timeUnit
	return opts
}

// Delay setter
func (opts RetryOpts) Delay(delay time.Duration) RetryOpts {
	opts.delay = delay
	return opts
}

// Tries setter
func (opts RetryOpts) Tries(tries uint) RetryOpts {
	opts.tries = tries
	return opts
}
