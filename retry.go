// Simple library for retry mechanism
// slightly inspired by https://metacpan.org/pod/Try::Tiny::Retry
package retry

import (
	"errors"
	"fmt"
	"time"
)

type Retryable func() error
type OnRetry func(n uint, err error)

var defaultTries uint = 10
var defaultDelay time.Duration = 1e5

// Retry - simple retry
//
//	url := "http://example.com"
//	var body []byte
//
//	err := retry.Retry(
//		func() error {
//			resp, err := http.Get(url)
//			if err != nil {
//				return err
//			}
//			defer resp.Body.Close()
//			body, err = ioutil.ReadAll(resp.Body)
//			if err != nil {
//				return err
//			}
//
//			return nil
//		},
//	)
func Retry(retryableFunction Retryable) error {
	return RetryWithOpts(retryableFunction, NewRetryOpts())
}

// RetryWithOpts - customizable retry via RetryOpts
func RetryWithOpts(retryableFunction Retryable, opts RetryOpts) error {
	return RetryCustom(retryableFunction, func(n uint, err error) {}, opts)
}

// RetryCustom - the most customizable retry
// is possible set OnRetry function callback
// which are called each retry
func RetryCustom(retryableFunction Retryable, onRetryFunction OnRetry, opts RetryOpts) error {
	var n uint = 0

	for n < opts.tries {
		err := retryableFunction()

		if err != nil {
			onRetryFunction(n, err)

			delayTime := opts.delay * (1 << (n - 1))
			time.Sleep((time.Duration)(delayTime) * opts.units)
		} else {
			return nil
		}

		n++
	}

	return errors.New(fmt.Sprintf("All (%d) retries fail", opts.tries))
}
