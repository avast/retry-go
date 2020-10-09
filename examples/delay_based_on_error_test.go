// This test delay is based on kind of error
// e.g. HTTP response [Retry-After](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Retry-After)
package retry_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/avast/retry-go"
	"github.com/stretchr/testify/assert"
)

func TestParseRetryAfter(t *testing.T) {
	retryAfter, err := ParseRetryAfter("xyz")
	assert.Error(t, err)

	retryAfter, err = ParseRetryAfter("")
	assert.Error(t, err)

	testDur := 120 * time.Second

	retryAfter, err = ParseRetryAfter("120")
	assert.NoError(t, err)
	assert.Equal(t, testDur, retryAfter, "time in seconds")

	retryAfter, err = ParseRetryAfter(time.Now().Add(testDur).Format(time.RFC850))
	assert.NoError(t, err)
	assert.True(t, retryAfter > (115*time.Second), "time in seconds are ~120")
	t.Log(retryAfter)
}

func ParseRetryAfter(ra string) (time.Duration, error) {
	if ra == "" {
		return 0, fmt.Errorf("Retry-After header was empty")
	}

	t, errParse := http.ParseTime(ra)
	if errParse != nil {
		if sec, errParse := strconv.Atoi(ra); errParse == nil {
			return time.Duration(sec) * time.Second, nil
		}
	} else {
		return t.Sub(time.Now()), nil
	}

	return 0, fmt.Errorf("Invalid Retr-After format %s", ra)
}

type RetryAfterError struct {
	response http.Response
}

func (err RetryAfterError) Error() string {
	return fmt.Sprintf("Request to %s fail %s (%d)", err.response.Request.RequestURI, err.response.Status, err.response.StatusCode)
}

type SomeOtherError struct {
	err        string
	retryAfter time.Duration
}

func (err SomeOtherError) Error() string {
	return err.err
}

func TestCustomRetryFunctionBasedOnKindOfError(t *testing.T) {
	url := "http://example.com"
	var body []byte

	err := retry.Do(
		func() error {
			resp, err := http.Get(url)

			if err == nil {
				defer func() {
					if err := resp.Body.Close(); err != nil {
						panic(err)
					}
				}()
				body, err = ioutil.ReadAll(resp.Body)
			}

			return err
		},
		retry.DelayType(func(n uint, err error, config *retry.Config) time.Duration {
			switch err.(type) {
			case RetryAfterError:
				e := err.(RetryAfterError)
				if dur, err := ParseRetryAfter(e.response.Header.Get("Retry-After")); err == nil {
					return dur
				}
			case SomeOtherError:
				e := err.(SomeOtherError)
				return e.retryAfter
			}

			//default is backoffdelay
			return retry.BackOffDelay(n, err, config)
		}),
	)

	assert.NoError(t, err)
	assert.NotEmpty(t, body)
}
