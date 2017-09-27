// Simple library for retry mechanism
// slightly inspired by https://metacpan.org/pod/Try::Tiny::Retry
package retry

import (
	"fmt"
	"strings"
	"time"
)

// Function signature of retryable function
type Retryable func() error

// Function signature of OnRetry function
// n = count of tries
type OnRetry func(n uint, err error)

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
	var n uint

	errorLog := make(errorLog, opts.tries)

	for n < opts.tries {
		err := retryableFunction()

		if err != nil {
			onRetryFunction(n, err)
			errorLog[n] = err

			delayTime := opts.delay * (1 << (n - 1))
			time.Sleep((time.Duration)(delayTime) * opts.units)
		} else {
			return nil
		}

		n++
	}

	return fmt.Errorf("All (%d) retries fail:\n%s", opts.tries, errorLog)
}

type errorLog []error

func (log errorLog) String() string {
	logWithNumber := make([]string, len(log))
	for i, l := range log {
		logWithNumber[i] = fmt.Sprintf("#%d: %s", i+1, l.Error())
	}

	return strings.Join(logWithNumber, "\n")
}
