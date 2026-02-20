package processor

import "context"

// semaphore implements a simple counting semaphore for limiting concurrency
type semaphore struct {
	ch chan struct{}
}

// newSemaphore creates a new semaphore with the given capacity
func newSemaphore(capacity int) *semaphore {
	return &semaphore{
		ch: make(chan struct{}, capacity),
	}
}

// acquire acquires a semaphore slot, blocking if necessary
func (s *semaphore) acquire(ctx context.Context) error {
	select {
	case s.ch <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// release releases a semaphore slot
func (s *semaphore) release() {
	<-s.ch
}
