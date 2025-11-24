package retry_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/avast/retry-go/v5"
	"github.com/stretchr/testify/assert"
)

// TestErrorHistory shows an example of how to get all the previous errors when
// retry.Do ends in success
func TestErrorHistory(t *testing.T) {
	attempts := 3 // server succeeds after 3 attempts
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts > 0 {
			attempts--
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()
	var allErrors []error
	err := retry.New(
		retry.OnRetry(func(n uint, err error) {
			allErrors = append(allErrors, err)
		}),
	).Do(
		func() error {
			resp, err := http.Get(ts.URL)
			if err != nil {
				return err
			}
			// nolint:errcheck
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				return fmt.Errorf("failed HTTP - %d", resp.StatusCode)
			}
			return nil
		},
	)
	assert.NoError(t, err)
	assert.Len(t, allErrors, 3)
}
