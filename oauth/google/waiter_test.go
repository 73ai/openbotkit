package google

import (
	"errors"
	"testing"
	"time"
)

func TestScopeWaiter_SignalBeforeTimeout(t *testing.T) {
	w := NewScopeWaiter()

	done := make(chan error, 1)
	go func() {
		done <- w.Wait("state-1", 5*time.Second)
	}()

	time.Sleep(10 * time.Millisecond)
	w.Signal("state-1", nil)

	if err := <-done; err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestScopeWaiter_Timeout(t *testing.T) {
	w := NewScopeWaiter()

	err := w.Wait("state-2", 50*time.Millisecond)
	if !errors.Is(err, ErrAuthTimeout) {
		t.Fatalf("expected ErrAuthTimeout, got %v", err)
	}
}

func TestScopeWaiter_SignalWithError(t *testing.T) {
	w := NewScopeWaiter()
	want := errors.New("exchange failed")

	done := make(chan error, 1)
	go func() {
		done <- w.Wait("state-3", 5*time.Second)
	}()

	time.Sleep(10 * time.Millisecond)
	w.Signal("state-3", want)

	if err := <-done; err != want {
		t.Fatalf("expected %v, got %v", want, err)
	}
}

func TestScopeWaiter_SignalUnknownState(t *testing.T) {
	w := NewScopeWaiter()
	// Should not panic.
	w.Signal("unknown", nil)
}
