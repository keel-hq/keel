// Package concurrent provides small helpers for running bounded
// concurrency work items, such as fanning out per-deployment updates.
package concurrent

import "sync"

// Run invokes fn for every index in [0, n) using at most limit goroutines
// in flight at any time. Items may complete out of order; Run blocks until
// every item has been processed. A limit <= 0 is treated as 1.
func Run(limit int, n int, fn func(index int)) {
	if n <= 0 {
		return
	}
	if limit <= 0 {
		limit = 1
	}
	if limit > n {
		limit = n
	}

	jobs := make(chan int, limit)
	var wg sync.WaitGroup
	for w := 0; w < limit; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				fn(i)
			}
		}()
	}

	for i := 0; i < n; i++ {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
}
