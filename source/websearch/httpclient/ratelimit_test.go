package httpclient

import (
	"context"
	"testing"
	"time"
)

func TestRateLimiterDifferentHostsIndependent(t *testing.T) {
	rl := newHostRateLimiter()
	ctx := context.Background()

	start := time.Now()
	for range 3 {
		if err := rl.Wait(ctx, "a.com"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := rl.Wait(ctx, "b.com"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	elapsed := time.Since(start)

	// Burst of 3 per host means the first 3 requests to each host
	// should be nearly instant.
	if elapsed > 500*time.Millisecond {
		t.Errorf("burst requests should be fast, took %v", elapsed)
	}
}

func TestRateLimiterContextCancellation(t *testing.T) {
	rl := newHostRateLimiter()
	ctx, cancel := context.WithCancel(context.Background())

	// Exhaust the burst.
	for range 3 {
		rl.Wait(ctx, "slow.com")
	}

	cancel()
	err := rl.Wait(ctx, "slow.com")
	if err == nil {
		t.Error("expected error on cancelled context")
	}
}
