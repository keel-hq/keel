package stopper

import (
	"context"
	"sync"
	"time"
)

// Stopper eases the graceful termination of a group of goroutines
type Stopper struct {
	ctx  context.Context
	wg   sync.WaitGroup
	stop chan struct{}
}

// NewStopper initializes a new Stopper instance
func NewStopper(ctx context.Context) *Stopper {
	return &Stopper{ctx: ctx}
}

// Begin indicates that a new goroutine has started.
func (s *Stopper) Begin() {
	s.wg.Add(1)
}

// End indicates that a goroutine has stopped.
func (s *Stopper) End() {
	s.wg.Done()
}

// Chan returns the channel on which goroutines could listen to determine if
// they should stop. The channel is closed when Stop() is called.
func (s *Stopper) Chan() chan struct{} {
	return s.stop
}

// Sleep puts the current goroutine on sleep during a duration d
// Sleep could be interrupted in the case the goroutine should stop itself,
// in which case Sleep returns false.
func (s *Stopper) Sleep(d time.Duration) bool {
	select {
	case <-time.After(d):
		return true
	case <-s.ctx.Done():
		return false
	}
}

// Stop asks every goroutine to end.
func (s *Stopper) Stop() {
	close(s.stop)
	s.wg.Wait()
}
