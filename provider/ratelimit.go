package provider

import (
	"context"

	"golang.org/x/time/rate"
)

// RateLimiter throttles LLM API calls using a token bucket.
type RateLimiter struct {
	limiter *rate.Limiter
}

// NewRateLimiter creates a rate limiter allowing requestsPerHour with a burst of 10.
func NewRateLimiter(requestsPerHour int) *RateLimiter {
	r := float64(requestsPerHour) / 3600.0
	return &RateLimiter{limiter: rate.NewLimiter(rate.Limit(r), 10)}
}

// Wait blocks until the rate limiter allows one more event, or ctx is canceled.
func (r *RateLimiter) Wait(ctx context.Context) error {
	return r.limiter.Wait(ctx)
}
