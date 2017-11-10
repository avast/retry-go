/*
Simple library for retry mechanism

slightly inspired by [Try::Tiny::Retry](https://metacpan.org/pod/Try::Tiny::Retry)

SYNOPSIS

http get with retry:

	url := "http://example.com"
	var body []byte

	err := retry.Retry(
		func() error {
			resp, err := http.Get(url)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			body, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			return nil
		},
	)

	fmt.Println(body)

[next examples](https://github.com/avast/retry-go/examples)


SEE ALSO

* [giantswarm/retry-go](https://github.com/giantswarm/retry-go) - slightly complicated interface.

* [sethgrid/pester](https://github.com/sethgrid/pester) - only http retry for http calls with retries and backoff

* [cenkalti/backoff](https://github.com/cenkalti/backoff) - Go port of the exponential backoff algorithm from Google's HTTP Client Library for Java. Really complicated interface.

* [rafaeljesus/retry-go](https://github.com/rafaeljesus/retry-go) - looks good, slightly similar as this package, don't have 'simple' `Retry` method

* [matryer/try](https://github.com/matryer/try) - very popular package, nonintuitive interface (for me)

*/
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

	errorLog := make(Error, opts.tries)

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

	return errorLog
}

// Error type represents list of errors in retry
type Error []error

// Error method return string representation of Error
// It is an implementation of error interface
func (e Error) Error() string {
	logWithNumber := make([]string, len(e))
	for i, l := range e {
		logWithNumber[i] = fmt.Sprintf("#%d: %s", i+1, l.Error())
	}

	return fmt.Sprintf("All retries fail:\n%s", strings.Join(logWithNumber, "\n"))
}

// WrappedErrors returns the list of errors that this Error is wrapping.
// It is an implementation of the `errwrap.Wrapper` interface
// in package [errwrap](https://github.com/hashicorp/errwrap) so that
// `retry.Error` can be used with that library.
func (e Error) WrappedErrors() []error {
	return e
}
