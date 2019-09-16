package rest

import (
	"fmt"
	"time"
)

// Retry implements boiler plate code to easily call the same function until it suceeds
// or a time-out is reached. User can define: (1) the interval between two calls (default 1 second) and
// (2) the delay before timeout (default 30s). You can define none of them or the interval by simply passing
// zero, one or two time.Duration values.
func Retry(f func() error, seconds ...time.Duration) error {

	if err := f(); err == nil {
		return nil
	}

	tick := time.Tick(1 * time.Second)
	timeout := time.After(30 * time.Second)
	if len(seconds) == 2 {
		tick = time.Tick(seconds[0])
		timeout = time.After(seconds[1])
	} else if len(seconds) == 1 {
		tick = time.Tick(seconds[0])
	}

	for {
		select {
		case <-tick:
			if err := f(); err == nil {
				return nil
			}
		case <-timeout:
			return fmt.Errorf("timeout")
		}
	}
}
