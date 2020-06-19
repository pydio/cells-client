package rest

import (
	"time"
)

// RetryCallback implements boiler plate code to easily call the same function until it suceeds
// or a time-out is reached.
func RetryCallback(callback func() error, number int, interval time.Duration) error {

	var e error
	for i := 0; i < number; i++ {
		if e = callback(); e == nil {
			break
		}
		if i < number-1 {
			<-time.After(interval)
		}
	}

	return e
}
