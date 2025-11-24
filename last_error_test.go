package retry

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestLastError verifies the LastError helper method
func TestLastError(t *testing.T) {
	err1 := errors.New("error 1")
	err2 := errors.New("error 2")
	err3 := errors.New("error 3")

	t.Run("returns last error from list", func(t *testing.T) {
		retryErr := Error{err1, err2, err3}
		last := retryErr.LastError()
		assert.Equal(t, err3, last, "LastError should return the last error")
	})

	t.Run("returns nil for empty error list", func(t *testing.T) {
		emptyErr := Error{}
		assert.Nil(t, emptyErr.LastError(), "LastError should return nil for empty error list")
	})

	t.Run("returns only error for single error", func(t *testing.T) {
		singleErr := Error{err1}
		assert.Equal(t, err1, singleErr.LastError(), "LastError should return the only error")
	})

	t.Run("migration example from errors.Unwrap", func(t *testing.T) {
		// Simulate a retry that failed 3 times
		retryErr := New(Attempts(3), Delay(0)).Do(func() error {
			return errors.New("operation failed")
		})

		// Old v4.x way (no longer works):
		// lastErr := errors.Unwrap(retryErr)

		// New v5.0.0 way - option 1 (recommended):
		// Can't use errors.Is with dynamic error, so check manually
		assert.Error(t, retryErr)

		// New v5.0.0 way - option 2 (if you need the last error):
		if e, ok := retryErr.(Error); ok {
			lastErr := e.LastError()
			assert.NotNil(t, lastErr)
			assert.Contains(t, lastErr.Error(), "operation failed")
		} else {
			t.Fatal("Expected retryErr to be retry.Error")
		}
	})
}
