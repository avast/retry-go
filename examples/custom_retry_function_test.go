package retry_test

import (
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/avast/retry-go"
	"github.com/stretchr/testify/assert"
)

func TestCustomRetryFunction(t *testing.T) {
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
		retry.DelayType(func(n uint, config *retry.Config) time.Duration {
			return 0
		}),
	)

	assert.NoError(t, err)
	assert.NotEmpty(t, body)
}
