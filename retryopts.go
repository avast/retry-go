package retry

import (
	"time"
)

type RetryOpts struct {
	tries uint
	delay time.Duration
	units time.Duration
}

func NewRetryOpts() RetryOpts {
	return RetryOpts{tries: defaultTries, delay: defaultDelay, units: time.Microsecond}
}

func (opts RetryOpts) Units(timeUnit time.Duration) RetryOpts {
	opts.units = timeUnit
	return opts
}

func (opts RetryOpts) Delay(delay time.Duration) RetryOpts {
	opts.delay = delay
	return opts
}

func (opts RetryOpts) Tries(tries uint) RetryOpts {
	opts.tries = tries
	return opts
}
