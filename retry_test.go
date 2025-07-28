package retry

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDoWithDataAllFailed(t *testing.T) {
	var retrySum uint
	v, err := DoWithData(
		func() (int, error) { return 7, errors.New("test") },
		OnRetry(func(n uint, err error) { retrySum += n }),
		Delay(time.Nanosecond),
	)
	assert.Error(t, err)
	assert.Equal(t, 0, v)

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
	assert.Len(t, err, 10)
	fmt.Println(err.Error())
	assert.Equal(t, expectedErrorFormat, err.Error(), "retry error format")
	assert.Equal(t, uint(36), retrySum, "right count of retry")
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

func TestDoWithDataFirstOk(t *testing.T) {
	returnVal := 1

	var retrySum uint
	val, err := DoWithData(
		func() (int, error) { return returnVal, nil },
		OnRetry(func(n uint, err error) { retrySum += n }),
	)
	assert.NoError(t, err)
	assert.Equal(t, returnVal, val)
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
	assert.Len(t, err, 3)
	assert.Equal(t, expectedErrorFormat, err.Error(), "retry error format")
	assert.Equal(t, uint(2), retryCount, "right count of retry")
}

func TestRetryIf_ZeroAttempts(t *testing.T) {
	var retryCount, onRetryCount uint
	err := Do(
		func() error {
			if retryCount >= 2 {
				return errors.New("special")
			} else {
				retryCount++
				return errors.New("test")
			}
		},
		OnRetry(func(n uint, err error) { onRetryCount = n }),
		RetryIf(func(err error) bool {
			return err.Error() != "special"
		}),
		Delay(time.Nanosecond),
		Attempts(0),
	)
	assert.Error(t, err)

	assert.Equal(t, "special", err.Error(), "retry error format")
	assert.Equal(t, retryCount, onRetryCount+1, "right count of retry")
}

func TestZeroAttemptsWithError(t *testing.T) {
	const maxErrors = 999
	count := 0

	err := Do(
		func() error {
			if count < maxErrors {
				count += 1
				return errors.New("test")
			}

			return nil
		},
		Attempts(0),
		MaxDelay(time.Nanosecond),
	)
	assert.NoError(t, err)

	assert.Equal(t, count, maxErrors)
}

func TestZeroAttemptsWithoutError(t *testing.T) {
	count := 0

	err := Do(
		func() error {
			count++

			return nil
		},
		Attempts(0),
	)
	assert.NoError(t, err)

	assert.Equal(t, count, 1)
}

func TestZeroAttemptsWithUnrecoverableError(t *testing.T) {
	err := Do(
		func() error {
			return Unrecoverable(assert.AnError)
		},
		Attempts(0),
		MaxDelay(time.Nanosecond),
	)
	assert.Error(t, err)
	assert.Equal(t, Unrecoverable(assert.AnError), err)
}

func TestAttemptsForError(t *testing.T) {
	count := uint(0)
	testErr := os.ErrInvalid
	attemptsForTestError := uint(3)
	err := Do(
		func() error {
			count++
			return testErr
		},
		AttemptsForError(attemptsForTestError, testErr),
		Attempts(5),
	)
	assert.Error(t, err)
	assert.Equal(t, attemptsForTestError, count)
}

func TestDefaultSleep(t *testing.T) {
	start := time.Now()
	err := Do(
		func() error { return errors.New("test") },
		Attempts(3),
	)
	dur := time.Since(start)
	assert.Error(t, err)
	assert.Greater(t, dur, 300*time.Millisecond, "3 times default retry is longer then 300ms")
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
	assert.Less(t, dur, 500*time.Millisecond, "3 times default retry is shorter then 500ms")
}

func TestLastErrorOnly(t *testing.T) {
	var retrySum uint
	err := Do(
		func() error { return fmt.Errorf("%d", retrySum) },
		OnRetry(func(n uint, err error) { retrySum += 1 }),
		Delay(time.Nanosecond),
		LastErrorOnly(true),
	)
	assert.Error(t, err)
	assert.Equal(t, "9", err.Error())
}

func TestUnrecoverableError(t *testing.T) {
	attempts := 0
	testErr := errors.New("error")
	expectedErr := Error{testErr}
	err := Do(
		func() error {
			attempts++
			return Unrecoverable(testErr)
		},
		Attempts(2),
	)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, testErr, errors.Unwrap(err))
	assert.Equal(t, 1, attempts, "unrecoverable error broke the loop")
}

func TestCombineFixedDelays(t *testing.T) {
	if os.Getenv("OS") == "macos-latest" {
		t.Skip("Skipping testing in MacOS GitHub actions - too slow, duration is wrong")
	}

	start := time.Now()
	err := Do(
		func() error { return errors.New("test") },
		Attempts(3),
		DelayType(CombineDelay(FixedDelay, FixedDelay)),
	)
	dur := time.Since(start)
	assert.Error(t, err)
	assert.Greater(t, dur, 400*time.Millisecond, "3 times combined, fixed retry is greater then 400ms")
	assert.Less(t, dur, 500*time.Millisecond, "3 times combined, fixed retry is less then 500ms")
}

func TestRandomDelay(t *testing.T) {
	if os.Getenv("OS") == "macos-latest" {
		t.Skip("Skipping testing in MacOS GitHub actions - too slow, duration is wrong")
	}

	start := time.Now()
	err := Do(
		func() error { return errors.New("test") },
		Attempts(3),
		DelayType(RandomDelay),
		MaxJitter(50*time.Millisecond),
	)
	dur := time.Since(start)
	assert.Error(t, err)
	assert.Greater(t, dur, 2*time.Millisecond, "3 times random retry is longer then 2ms")
	assert.Less(t, dur, 150*time.Millisecond, "3 times random retry is shorter then 150ms")
}

func TestMaxDelay(t *testing.T) {
	if os.Getenv("OS") == "macos-latest" {
		t.Skip("Skipping testing in MacOS GitHub actions - too slow, duration is wrong")
	}

	start := time.Now()
	err := Do(
		func() error { return errors.New("test") },
		Attempts(5),
		Delay(10*time.Millisecond),
		MaxDelay(50*time.Millisecond),
	)
	dur := time.Since(start)
	assert.Error(t, err)
	assert.Greater(t, dur, 120*time.Millisecond, "5 times with maximum delay retry is less than 120ms")
	assert.Less(t, dur, 275*time.Millisecond, "5 times with maximum delay retry is longer than 275ms")
}

func TestBackOffDelay(t *testing.T) {
	for _, c := range []struct {
		label         string
		delay         time.Duration
		expectedMaxN  uint
		n             uint
		expectedDelay time.Duration
	}{
		{
			label:         "negative-delay",
			delay:         -1,
			expectedMaxN:  62,
			n:             2,
			expectedDelay: 4,
		},
		{
			label:         "zero-delay",
			delay:         0,
			expectedMaxN:  62,
			n:             65,
			expectedDelay: 1 << 62,
		},
		{
			label:         "one-second",
			delay:         time.Second,
			expectedMaxN:  33,
			n:             62,
			expectedDelay: time.Second << 33,
		},
	} {
		t.Run(
			c.label,
			func(t *testing.T) {
				config := Config{
					delay: c.delay,
				}
				delay := BackOffDelay(c.n, nil, &config)
				assert.Equal(t, c.expectedMaxN, config.maxBackOffN, "max n mismatch")
				assert.Equal(t, c.expectedDelay, delay, "delay duration mismatch")
			},
		)
	}
}

func TestCombineDelay(t *testing.T) {
	f := func(d time.Duration) DelayTypeFunc {
		return func(_ uint, _ error, _ *Config) time.Duration {
			return d
		}
	}
	const max = time.Duration(1<<63 - 1)
	for _, c := range []struct {
		label    string
		delays   []time.Duration
		expected time.Duration
	}{
		{
			label: "empty",
		},
		{
			label: "single",
			delays: []time.Duration{
				time.Second,
			},
			expected: time.Second,
		},
		{
			label: "negative",
			delays: []time.Duration{
				time.Second,
				-time.Millisecond,
			},
			expected: time.Second - time.Millisecond,
		},
		{
			label: "overflow",
			delays: []time.Duration{
				max,
				time.Second,
				time.Millisecond,
			},
			expected: max,
		},
	} {
		t.Run(
			c.label,
			func(t *testing.T) {
				funcs := make([]DelayTypeFunc, len(c.delays))
				for i, d := range c.delays {
					funcs[i] = f(d)
				}
				actual := CombineDelay(funcs...)(0, nil, nil)
				assert.Equal(t, c.expected, actual, "delay duration mismatch")
			},
		)
	}
}

func TestContext(t *testing.T) {
	const defaultDelay = 100 * time.Millisecond
	t.Run("cancel before", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		retrySum := 0
		start := time.Now()
		err := Do(
			func() error { return errors.New("test") },
			OnRetry(func(n uint, err error) { retrySum += 1 }),
			Context(ctx),
		)
		dur := time.Since(start)
		assert.Error(t, err)
		assert.True(t, dur < defaultDelay, "immediately cancellation")
		assert.Equal(t, 0, retrySum, "called at most once")
	})

	t.Run("cancel in retry progress", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		retrySum := 0
		err := Do(
			func() error { return errors.New("test") },
			OnRetry(func(n uint, err error) {
				retrySum += 1
				if retrySum > 1 {
					cancel()
				}
			}),
			Context(ctx),
		)
		assert.Error(t, err)

		expectedErrorFormat := `All attempts fail:
#1: test
#2: test
#3: context canceled`
		assert.Len(t, err, 3)
		assert.Equal(t, expectedErrorFormat, err.Error(), "retry error format")
		assert.Equal(t, 2, retrySum, "called at most once")
	})

	t.Run("cancel in retry progress - last error only", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		retrySum := 0
		err := Do(
			func() error { return errors.New("test") },
			OnRetry(func(n uint, err error) {
				retrySum += 1
				if retrySum > 1 {
					cancel()
				}
			}),
			Context(ctx),
			LastErrorOnly(true),
		)
		assert.Equal(t, context.Canceled, err)

		assert.Equal(t, 2, retrySum, "called at most once")
	})

	t.Run("cancel in retry progress - infinite attempts", func(t *testing.T) {
		go func() {
			ctx, cancel := context.WithCancel(context.Background())

			retrySum := 0
			err := Do(
				func() error { return errors.New("test") },
				OnRetry(func(n uint, err error) {
					fmt.Println(n)
					retrySum += 1
					if retrySum > 1 {
						cancel()
					}
				}),
				Context(ctx),
				Attempts(0),
			)

			assert.Equal(t, context.Canceled, err)

			assert.Equal(t, 2, retrySum, "called at most once")
		}()
	})

	t.Run("cancelled on retry infinte attempts - wraps context error with last retried function error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		retrySum := 0
		err := Do(
			func() error { return fooErr{str: fmt.Sprintf("error %d", retrySum+1)} },
			OnRetry(func(n uint, err error) {
				retrySum += 1
				if retrySum == 2 {
					cancel()
				}
			}),
			Context(ctx),
			Attempts(0),
			WrapContextErrorWithLastError(true),
		)
		assert.ErrorIs(t, err, context.Canceled)
		assert.ErrorIs(t, err, fooErr{str: "error 2"})
	})

	t.Run("timed out on retry infinte attempts - wraps context error with last retried function error", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
		defer cancel()

		retrySum := 0
		err := Do(
			func() error { return fooErr{str: fmt.Sprintf("error %d", retrySum+1)} },
			OnRetry(func(n uint, err error) {
				retrySum += 1
			}),
			Context(ctx),
			Attempts(0),
			WrapContextErrorWithLastError(true),
		)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
		assert.ErrorIs(t, err, fooErr{str: "error 2"})
	})
}

type testTimer struct {
	called bool
}

func (t *testTimer) After(d time.Duration) <-chan time.Time {
	t.called = true
	return time.After(d)
}

func TestTimerInterface(t *testing.T) {
	var timer testTimer
	err := Do(
		func() error { return errors.New("test") },
		Attempts(1),
		Delay(10*time.Millisecond),
		MaxDelay(50*time.Millisecond),
		WithTimer(&timer),
	)

	assert.Error(t, err)

}

func TestErrorIs(t *testing.T) {
	var e Error
	expectErr := errors.New("error")
	closedErr := os.ErrClosed
	e = append(e, expectErr)
	e = append(e, closedErr)

	assert.True(t, errors.Is(e, expectErr))
	assert.True(t, errors.Is(e, closedErr))
	assert.False(t, errors.Is(e, errors.New("error")))
}

type fooErr struct{ str string }

func (e fooErr) Error() string {
	return e.str
}

type barErr struct{ str string }

func (e barErr) Error() string {
	return e.str
}

func TestErrorAs(t *testing.T) {
	var e Error
	fe := fooErr{str: "foo"}
	e = append(e, fe)

	var tf fooErr
	var tb barErr

	assert.True(t, errors.As(e, &tf))
	assert.False(t, errors.As(e, &tb))
	assert.Equal(t, "foo", tf.str)
}

func TestUnwrap(t *testing.T) {
	testError := errors.New("test error")
	err := Do(
		func() error {
			return testError
		},
		Attempts(1),
	)

	assert.Error(t, err)
	assert.Equal(t, testError, errors.Unwrap(err))
}

func BenchmarkDo(b *testing.B) {
	testError := errors.New("test error")

	for i := 0; i < b.N; i++ {
		_ = Do(
			func() error {
				return testError
			},
			Attempts(10),
			Delay(0),
		)
	}
}

func BenchmarkDoWithData(b *testing.B) {
	testError := errors.New("test error")

	for i := 0; i < b.N; i++ {
		_, _ = DoWithData(
			func() (int, error) {
				return 0, testError
			},
			Attempts(10),
			Delay(0),
		)
	}
}

func BenchmarkDoNoErrors(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Do(
			func() error {
				return nil
			},
			Attempts(10),
			Delay(0),
		)
	}
}

func BenchmarkDoWithDataNoErrors(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = DoWithData(
			func() (int, error) {
				return 0, nil
			},
			Attempts(10),
			Delay(0),
		)
	}
}

func TestOnRetryNotCalledOnLastAttempt(t *testing.T) {
	callCount := 0
	onRetryCalls := make([]uint, 0)
	
	err := Do(
		func() error {
			callCount++
			return errors.New("test error")
		},
		Attempts(3),
		OnRetry(func(n uint, err error) {
			onRetryCalls = append(onRetryCalls, n)
		}),
		Delay(time.Nanosecond),
	)
	
	assert.Error(t, err)
	assert.Equal(t, 3, callCount, "function should be called 3 times")
	assert.Equal(t, []uint{0, 1}, onRetryCalls, "onRetry should only be called for first 2 attempts, not the final one")
	assert.Len(t, onRetryCalls, 2, "onRetry should be called exactly 2 times (not on last attempt)")
}

func TestIsRecoverable(t *testing.T) {
	err := errors.New("err")
	assert.True(t, IsRecoverable(err))

	err = Unrecoverable(err)
	assert.False(t, IsRecoverable(err))

	err = fmt.Errorf("wrapping: %w", err)
	assert.False(t, IsRecoverable(err))
}
