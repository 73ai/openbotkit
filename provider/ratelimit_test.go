package provider

import (
	"context"
	"testing"
	"time"
)

func TestRateLimiter_BurstAllowed(t *testing.T) {
	rl := NewRateLimiter(3600) // 1 per second, burst of 10

	// Burst of 10 should succeed immediately.
	for i := range 10 {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		if err := rl.Wait(ctx); err != nil {
			t.Fatalf("burst call %d: %v", i, err)
		}
		cancel()
	}
}

func TestRateLimiter_ContextCanceled(t *testing.T) {
	rl := NewRateLimiter(1) // 1 per hour — extremely slow

	// Exhaust the burst.
	for range 10 {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		rl.Wait(ctx)
		cancel()
	}

	// Next call should block and be canceled.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := rl.Wait(ctx)
	if err == nil {
		t.Fatal("expected context deadline exceeded")
	}
}
