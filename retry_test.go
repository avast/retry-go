package retry

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestCustom(t *testing.T) {
	var retrySum uint
	err := RetryCustom(
		func() error { return errors.New("test") },
		func(n uint, err error) { retrySum += n },
		NewRetryOpts().Units(time.Nanosecond),
	)
	assert.Error(t, err)
	t.Log(err)
	assert.Equal(t, uint(45), retrySum, "right count of retry")

	retrySum = 0
	err = RetryCustom(
		func() error { return nil },
		func(n uint, err error) { retrySum += n },
		NewRetryOpts(),
	)
	assert.NoError(t, err)
	assert.Equal(t, uint(0), retrySum, "no retry")
}
