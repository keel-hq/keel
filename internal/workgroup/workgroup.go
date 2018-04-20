package workgroup

import "sync"

// Group manages a set of goroutines with related lifetimes.
type Group struct {
	fn []func(<-chan struct{})
}

// Add adds a function to the Group. Must be called before Run.
func (g *Group) Add(fn func(<-chan struct{})) {
	g.fn = append(g.fn, fn)
}

// Run exectues each function registered with Add in its own goroutine.
// Run blocks until each function has returned.
// The first function to return will trigger the closure of the channel
// passed to each function, who should in turn, return.
func (g *Group) Run() {
	var wg sync.WaitGroup
	wg.Add(len(g.fn))

	stop := make(chan struct{})
	result := make(chan error, len(g.fn))
	for _, fn := range g.fn {
		go func(fn func(<-chan struct{})) {
			defer wg.Done()
			fn(stop)
			result <- nil
		}(fn)
	}

	<-result
	close(stop)
	wg.Wait()
}
