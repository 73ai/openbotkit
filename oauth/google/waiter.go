package google

import (
	"errors"
	"sync"
	"time"
)

// ErrAuthTimeout is returned when a scope grant request times out.
var ErrAuthTimeout = errors.New("auth timeout: user did not complete OAuth in time")

type pendingAuth struct {
	ch      chan error
	scopes  []string
	account string
}

// ScopeWaiter coordinates between a tool that needs a scope grant
// and the OAuth callback that completes it.
type ScopeWaiter struct {
	mu      sync.Mutex
	pending map[string]*pendingAuth
}

func NewScopeWaiter() *ScopeWaiter {
	return &ScopeWaiter{pending: make(map[string]*pendingAuth)}
}

// Register adds a pending auth entry so that Signal can resolve it
// even before Await is called — eliminating the race where Signal
// fires between NotifyLink and Await.
func (w *ScopeWaiter) Register(state string, scopes []string, account string) {
	w.mu.Lock()
	w.pending[state] = &pendingAuth{
		ch:      make(chan error, 1),
		scopes:  scopes,
		account: account,
	}
	w.mu.Unlock()
}

// Await blocks until Signal is called for a previously registered state
// or the timeout expires.
func (w *ScopeWaiter) Await(state string, timeout time.Duration) error {
	w.mu.Lock()
	p, ok := w.pending[state]
	w.mu.Unlock()
	if !ok {
		return errors.New("state not registered")
	}

	defer func() {
		w.mu.Lock()
		delete(w.pending, state)
		w.mu.Unlock()
	}()

	select {
	case err := <-p.ch:
		return err
	case <-time.After(timeout):
		return ErrAuthTimeout
	}
}

// Wait registers a state and blocks until Signal is called or timeout.
// For race-free usage prefer Register + Await.
func (w *ScopeWaiter) Wait(state string, timeout time.Duration, scopes []string, account string) error {
	w.Register(state, scopes, account)
	return w.Await(state, timeout)
}

// Lookup returns the scopes and account associated with a pending state.
// Returns false if no pending auth exists for that state.
func (w *ScopeWaiter) Lookup(state string) (scopes []string, account string, ok bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	p, exists := w.pending[state]
	if !exists {
		return nil, "", false
	}
	return p.scopes, p.account, true
}

// Signal unblocks a waiting goroutine for the given state.
// If no goroutine is waiting for this state, Signal is a no-op.
func (w *ScopeWaiter) Signal(state string, err error) {
	w.mu.Lock()
	p, ok := w.pending[state]
	w.mu.Unlock()

	if ok {
		p.ch <- err
	}
}
