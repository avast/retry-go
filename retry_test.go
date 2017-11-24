package retry

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDo(t *testing.T) {
	var retrySum uint
	err := Do(
		func() error { return errors.New("test") },
		OnRetryFunction(func(n uint, err error) { retrySum += n }),
		Units(time.Nanosecond),
	)
	assert.Error(t, err)

	expectedErrorFormat := `All retries fail:
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
	err = Do(
		func() error { return nil },
		OnRetryFunction(func(n uint, err error) { retrySum += n }),
	)
	assert.NoError(t, err)
	assert.Equal(t, uint(0), retrySum, "no retry")

	var retryCount uint
	err = Do(
		func() error {
			if retryCount >= 2 {
				return errors.New("special")
			} else {
				return errors.New("test")
			}
		},
		OnRetryFunction(func(n uint, err error) { retryCount++ }),
		RetryIfFunction(func(err error) bool {
			return err.Error() != "special"
		}),
		Units(time.Nanosecond),
	)
	assert.Error(t, err)

	expectedErrorFormat = `All retries fail:
#1: test
#2: test
#3: special`
	assert.Equal(t, expectedErrorFormat, err.Error(), "retry error format")
	assert.Equal(t, uint(3), retryCount, "right count of retry")

}
