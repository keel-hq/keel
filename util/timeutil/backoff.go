package timeutil

import (
	"time"
)

// ExpBackoff - exponential backoff helper func
func ExpBackoff(prev, max time.Duration) time.Duration {
	if prev == 0 {
		return time.Second
	}
	if prev > max/2 {
		return max
	}
	return 2 * prev
}
