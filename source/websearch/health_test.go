package websearch

import (
	"testing"
	"time"
)

func TestHealthTrackerDefaultHealthy(t *testing.T) {
	h := newHealthTracker()
	if !h.IsHealthy("duckduckgo") {
		t.Error("unknown backend should be healthy")
	}
}

func TestHealthTrackerFailureCooldown(t *testing.T) {
	h := newHealthTracker()
	h.RecordFailure("brave", FailureTransient)

	if h.IsHealthy("brave") {
		t.Error("backend should be unhealthy right after failure")
	}
}

func TestHealthTrackerSuccessResets(t *testing.T) {
	h := newHealthTracker()
	h.RecordFailure("brave", FailureTransient)
	h.RecordSuccess("brave")

	if !h.IsHealthy("brave") {
		t.Error("backend should be healthy after success")
	}
}

func TestHealthTrackerCooldownDoubles(t *testing.T) {
	h := newHealthTracker()
	h.RecordFailure("brave", FailureTransient)

	h.mu.Lock()
	first := h.backends["brave"].cooldown
	h.mu.Unlock()

	h.RecordFailure("brave", FailureTransient)

	h.mu.Lock()
	second := h.backends["brave"].cooldown
	h.mu.Unlock()

	if second != first*2 {
		t.Errorf("expected cooldown to double: first=%v, second=%v", first, second)
	}
}

func TestHealthTrackerTransientCooldownCapped(t *testing.T) {
	h := newHealthTracker()
	for range 20 {
		h.RecordFailure("brave", FailureTransient)
	}

	h.mu.Lock()
	cooldown := h.backends["brave"].cooldown
	h.mu.Unlock()

	if cooldown > transientMaxCooldown {
		t.Errorf("cooldown should be capped at %v, got %v", transientMaxCooldown, cooldown)
	}
}

func TestHealthTrackerCooldownExpires(t *testing.T) {
	h := newHealthTracker()
	h.RecordFailure("brave", FailureTransient)

	// Backdate the failure past the cooldown.
	h.mu.Lock()
	h.backends["brave"].lastFail = time.Now().Add(-3 * transientMinCooldown)
	h.mu.Unlock()

	if !h.IsHealthy("brave") {
		t.Error("backend should be healthy after cooldown expires")
	}
}

func TestHealthTrackerIndependentBackends(t *testing.T) {
	h := newHealthTracker()
	h.RecordFailure("brave", FailureTransient)

	if !h.IsHealthy("duckduckgo") {
		t.Error("failure in brave should not affect duckduckgo")
	}
}

func TestHealthTrackerRateLimitCooldown(t *testing.T) {
	h := newHealthTracker()
	h.RecordFailure("duckduckgo", FailureRateLimit)

	h.mu.Lock()
	cooldown := h.backends["duckduckgo"].cooldown
	h.mu.Unlock()

	if cooldown < rateLimitMinCooldown {
		t.Errorf("rate limit cooldown should be at least %v, got %v", rateLimitMinCooldown, cooldown)
	}
}

func TestHealthTrackerAccessDeniedCooldown(t *testing.T) {
	h := newHealthTracker()
	h.RecordFailure("mojeek", FailureAccessDenied)

	h.mu.Lock()
	cooldown := h.backends["mojeek"].cooldown
	h.mu.Unlock()

	if cooldown != accessDeniedCooldown {
		t.Errorf("access denied cooldown should be %v, got %v", accessDeniedCooldown, cooldown)
	}
}

func TestHealthTrackerHalfOpenState(t *testing.T) {
	h := newHealthTracker()
	h.RecordFailure("brave", FailureTransient)

	// Backdate to expire the cooldown.
	h.mu.Lock()
	h.backends["brave"].lastFail = time.Now().Add(-2 * transientMinCooldown)
	h.mu.Unlock()

	// First call should transition to half-open and return healthy.
	if !h.IsHealthy("brave") {
		t.Error("backend should be healthy (half-open) after cooldown expires")
	}

	h.mu.Lock()
	state := h.backends["brave"].state
	h.mu.Unlock()

	if state != stateHalfOpen {
		t.Errorf("expected half-open state, got %d", state)
	}
}

func TestHealthTrackerHalfOpenFailureDoublesCooldown(t *testing.T) {
	h := newHealthTracker()
	h.RecordFailure("brave", FailureTransient)

	h.mu.Lock()
	firstCooldown := h.backends["brave"].cooldown
	h.backends["brave"].lastFail = time.Now().Add(-2 * transientMinCooldown)
	h.mu.Unlock()

	// Transition to half-open.
	h.IsHealthy("brave")

	// Fail again during half-open — should double the cooldown.
	h.RecordFailure("brave", FailureTransient)

	h.mu.Lock()
	newCooldown := h.backends["brave"].cooldown
	h.mu.Unlock()

	if newCooldown != firstCooldown*2 {
		t.Errorf("half-open failure should double cooldown: before=%v, after=%v", firstCooldown, newCooldown)
	}
}

func TestHealthTrackerHalfOpenSuccessRecovers(t *testing.T) {
	h := newHealthTracker()
	h.RecordFailure("brave", FailureTransient)

	// Backdate and transition to half-open.
	h.mu.Lock()
	h.backends["brave"].lastFail = time.Now().Add(-2 * transientMinCooldown)
	h.mu.Unlock()
	h.IsHealthy("brave")

	// Success during half-open should fully recover.
	h.RecordSuccess("brave")

	if !h.IsHealthy("brave") {
		t.Error("backend should be fully healthy after half-open success")
	}

	h.mu.Lock()
	_, exists := h.backends["brave"]
	h.mu.Unlock()

	if exists {
		t.Error("backend entry should be deleted after recovery")
	}
}

func TestHealthTrackerRateLimitCooldownCapped(t *testing.T) {
	h := newHealthTracker()
	for range 20 {
		h.RecordFailure("duckduckgo", FailureRateLimit)
	}

	h.mu.Lock()
	cooldown := h.backends["duckduckgo"].cooldown
	h.mu.Unlock()

	if cooldown > rateLimitMaxCooldown {
		t.Errorf("rate limit cooldown should be capped at %v, got %v", rateLimitMaxCooldown, cooldown)
	}
}
