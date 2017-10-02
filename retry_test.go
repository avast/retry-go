package retry

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCustom(t *testing.T) {
	var retrySum uint
	err := RetryCustom(
		func() error { return errors.New("test") },
		func(n uint, err error) { retrySum += n },
		NewRetryOpts().Units(time.Nanosecond),
	)
	assert.Error(t, err)

	expectedErrorFormat := `All (10) retries fail:
#1: test
#2: test
#3: test
#4: test
#5: test
#6: test
#7: test
#8: test
#9: test
#10: test`
	assert.Equal(t, expectedErrorFormat, err.Error(), "retry error format")
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
