package google

import (
	"errors"
	"sync"
	"time"
)

// ErrAuthTimeout is returned when a scope grant request times out.
var ErrAuthTimeout = errors.New("auth timeout: user did not complete OAuth in time")

// ScopeWaiter coordinates between a tool that needs a scope grant
// and the OAuth callback that completes it.
type ScopeWaiter struct {
	mu      sync.Mutex
	pending map[string]chan error
}

func NewScopeWaiter() *ScopeWaiter {
	return &ScopeWaiter{pending: make(map[string]chan error)}
}

// Wait blocks until Signal is called for the given state or the timeout expires.
func (w *ScopeWaiter) Wait(state string, timeout time.Duration) error {
	w.mu.Lock()
	ch := make(chan error, 1)
	w.pending[state] = ch
	w.mu.Unlock()

	defer func() {
		w.mu.Lock()
		delete(w.pending, state)
		w.mu.Unlock()
	}()

	select {
	case err := <-ch:
		return err
	case <-time.After(timeout):
		return ErrAuthTimeout
	}
}

// Signal unblocks a waiting goroutine for the given state.
// If no goroutine is waiting for this state, Signal is a no-op.
func (w *ScopeWaiter) Signal(state string, err error) {
	w.mu.Lock()
	ch, ok := w.pending[state]
	w.mu.Unlock()

	if ok {
		ch <- err
	}
}
