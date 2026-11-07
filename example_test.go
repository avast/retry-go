package retry_test

import (
	"errors"
	"fmt"
	"time"

	"github.com/avast/retry-go/v5"
)

type loggingTimer struct{}

func (t *loggingTimer) After(d time.Duration) <-chan time.Time {
	fmt.Println("Timer called!")
	return time.After(d)
}

func ExampleWithTimer() {
	attempts := 0
	err := retry.New(
		retry.WithTimer(&loggingTimer{}),
		retry.Attempts(2),
		retry.Delay(0),
		retry.DelayType(retry.FixedDelay),
	).Do(func() error {
		attempts++
		if attempts == 1 {
			return errors.New("try again")
		}
		return nil
	})

	fmt.Println("attempts:", attempts)
	fmt.Println("error:", err)
	// Output:
	// Timer called!
	// attempts: 2
	// error: <nil>
}
