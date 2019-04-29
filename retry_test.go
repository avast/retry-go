package retry

import (
	"errors"
	"testing"
	"time"

	"fmt"

	"github.com/stretchr/testify/assert"
)

func TestDoAllFailed(t *testing.T) {
	var retrySum uint
	err := Do(
		func() error { return errors.New("test") },
		OnRetry(func(n uint, err error) { retrySum += n }),
		Delay(time.Nanosecond),
	)
	assert.Error(t, err)

	expectedErrorFormat := `All attempts fail:
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
}

func TestDoFirstOk(t *testing.T) {
	var retrySum uint
	err := Do(
		func() error { return nil },
		OnRetry(func(n uint, err error) { retrySum += n }),
	)
	assert.NoError(t, err)
	assert.Equal(t, uint(0), retrySum, "no retry")

}

func TestRetryIf(t *testing.T) {
	var retryCount uint
	err := Do(
		func() error {
			if retryCount >= 2 {
				return errors.New("special")
			} else {
				return errors.New("test")
			}
		},
		OnRetry(func(n uint, err error) { retryCount++ }),
		RetryIf(func(err error) bool {
			return err.Error() != "special"
		}),
		Delay(time.Nanosecond),
	)
	assert.Error(t, err)

	expectedErrorFormat := `All attempts fail:
#1: test
#2: test
#3: special`
	assert.Equal(t, expectedErrorFormat, err.Error(), "retry error format")
	assert.Equal(t, uint(3), retryCount, "right count of retry")

}

func TestDefaultSleep(t *testing.T) {
	start := time.Now()
	err := Do(
		func() error { return errors.New("test") },
		Attempts(3),
	)
	dur := time.Since(start)
	assert.Error(t, err)
	assert.True(t, dur > 10*time.Millisecond, "3 times default retry is longer then 10ms")
}

func TestFixedSleep(t *testing.T) {
	start := time.Now()
	err := Do(
		func() error { return errors.New("test") },
		Attempts(3),
		DelayType(FixedDelay),
	)
	dur := time.Since(start)
	assert.Error(t, err)
	assert.True(t, dur < 500*time.Millisecond, "3 times default retry is shorter then 500ms")
}

func TestLastErrorOnly(t *testing.T) {
	var retrySum uint
	err := Do(
		func() error { return fmt.Errorf("%d", retrySum) },
		OnRetry(func(n uint, err error) { retrySum += 1 }),
		Delay(time.Nanosecond),
		LastErrorOnly(true),
	)
	assert.Equal(t, "9", err.Error())
}
