package retry_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/avast/retry-go/v4"
	"github.com/stretchr/testify/assert"
)

func TestGet(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintln(w, "hello")
	}))
	defer ts.Close()

	var body []byte

	err := retry.Do(
		func() error {
			resp, err := http.Get(ts.URL)

			if err == nil {
				defer func() {
					if err := resp.Body.Close(); err != nil {
						panic(err)
					}
				}()
				body, err = io.ReadAll(resp.Body)
			}

			return err
		},
	)

	assert.NoError(t, err)
	assert.NotEmpty(t, body)
}
