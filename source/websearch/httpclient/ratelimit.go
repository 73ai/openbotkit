package httpclient

import (
	"context"
	"sync"

	"golang.org/x/time/rate"
)

// TODO: evict stale entries if used in a long-lived process.
type hostRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	limit    rate.Limit
	burst    int
}

func newHostRateLimiter() *hostRateLimiter {
	return &hostRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		limit:    rate.Limit(1), // 1 request per second
		burst:    3,
	}
}

func (h *hostRateLimiter) Wait(ctx context.Context, host string) error {
	h.mu.Lock()
	lim, ok := h.limiters[host]
	if !ok {
		lim = rate.NewLimiter(h.limit, h.burst)
		h.limiters[host] = lim
	}
	h.mu.Unlock()

	return lim.Wait(ctx)
}
