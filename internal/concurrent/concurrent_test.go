package concurrent

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestRunProcessesEveryIndexExactlyOnce(t *testing.T) {
	const n = 64
	var seen atomic.Int64
	mu := sync.Mutex{}
	counts := make(map[int]int)

	Run(8, n, func(i int) {
		mu.Lock()
		counts[i]++
		mu.Unlock()
		seen.Add(1)
	})

	if seen.Load() != int64(n) {
		t.Fatalf("expected %d invocations, got %d", n, seen.Load())
	}
	for i := 0; i < n; i++ {
		if counts[i] != 1 {
			t.Fatalf("index %d processed %d times, want 1", i, counts[i])
		}
	}
}

func TestRunRespectsConcurrencyLimit(t *testing.T) {
	const (
		n     = 40
		limit = 4
	)
	var mu sync.Mutex
	var inFlight, maxInFlight, enteredCount int
	release := make(chan struct{})

	// The first `limit` items block until all of them are in flight, which
	// deterministically proves the limit is both reachable and not exceeded.
	Run(limit, n, func(_ int) {
		mu.Lock()
		inFlight++
		if inFlight > maxInFlight {
			maxInFlight = inFlight
		}
		enteredCount++
		allInFlight := enteredCount == limit
		mu.Unlock()
		if allInFlight {
			close(release)
		}
		<-release
		mu.Lock()
		inFlight--
		mu.Unlock()
	})

	if maxInFlight != limit {
		t.Fatalf("concurrency peaked at %d, want exactly %d", maxInFlight, limit)
	}
}

func TestRunEdgeCases(t *testing.T) {
	// n = 0: no invocations, must not deadlock.
	called := 0
	Run(4, 0, func(int) { called++ })
	if called != 0 {
		t.Fatalf("expected no invocations for n=0, got %d", called)
	}

	// limit > n: fine.
	done := 0
	Run(100, 3, func(int) { done++ })
	if done != 3 {
		t.Fatalf("expected 3 invocations, got %d", done)
	}

	// limit <= 0: treated as 1, still processes everything.
	done = 0
	Run(0, 5, func(int) { done++ })
	if done != 5 {
		t.Fatalf("expected 5 invocations with limit=0, got %d", done)
	}
}
