package provider

import (
	"context"
	"math/rand/v2"
	"time"
)

// ResilientProvider wraps a Provider with retry logic for transient errors.
type ResilientProvider struct {
	inner      Provider
	maxRetries int
	baseDelay  time.Duration
}

// NewResilientProvider wraps p with retry on retryable errors (3 retries, 1s base delay).
func NewResilientProvider(p Provider) *ResilientProvider {
	return &ResilientProvider{
		inner:      p,
		maxRetries: 3,
		baseDelay:  1 * time.Second,
	}
}

func (r *ResilientProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	var lastErr error
	for attempt := range r.maxRetries {
		resp, err := r.inner.Chat(ctx, req)
		if err == nil {
			return resp, nil
		}
		lastErr = err

		apiErr := ClassifyError(err)
		if apiErr.Kind != ErrorRetryable {
			return nil, err
		}

		if attempt < r.maxRetries-1 {
			if err := r.sleep(ctx, attempt); err != nil {
				return nil, lastErr
			}
		}
	}
	return nil, lastErr
}

func (r *ResilientProvider) StreamChat(ctx context.Context, req ChatRequest) (<-chan StreamEvent, error) {
	var lastErr error
	for attempt := range r.maxRetries {
		ch, err := r.inner.StreamChat(ctx, req)
		if err == nil {
			return ch, nil
		}
		lastErr = err

		apiErr := ClassifyError(err)
		if apiErr.Kind != ErrorRetryable {
			return nil, err
		}

		if attempt < r.maxRetries-1 {
			if err := r.sleep(ctx, attempt); err != nil {
				return nil, lastErr
			}
		}
	}
	return nil, lastErr
}

// sleep waits with exponential backoff and 25% jitter.
func (r *ResilientProvider) sleep(ctx context.Context, attempt int) error {
	delay := r.baseDelay * (1 << attempt)
	jitter := time.Duration(float64(delay) * 0.25 * rand.Float64())
	delay += jitter

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
