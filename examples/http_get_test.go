package retry_test

import (
	"fmt"
	"github.com/avast/retry-go"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"testing"
)

func TestGet(t *testing.T) {
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

	assert.NoError(t, err)
	assert.NotEmpty(t, body)
}
