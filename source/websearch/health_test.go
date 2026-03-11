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
	h.RecordFailure("brave")

	if h.IsHealthy("brave") {
		t.Error("backend should be unhealthy right after failure")
	}
}

func TestHealthTrackerSuccessResets(t *testing.T) {
	h := newHealthTracker()
	h.RecordFailure("brave")
	h.RecordSuccess("brave")

	if !h.IsHealthy("brave") {
		t.Error("backend should be healthy after success")
	}
}

func TestHealthTrackerCooldownDoubles(t *testing.T) {
	h := newHealthTracker()
	h.RecordFailure("brave")

	h.mu.Lock()
	first := h.backends["brave"].cooldown
	h.mu.Unlock()

	h.RecordFailure("brave")

	h.mu.Lock()
	second := h.backends["brave"].cooldown
	h.mu.Unlock()

	if second != first*2 {
		t.Errorf("expected cooldown to double: first=%v, second=%v", first, second)
	}
}

func TestHealthTrackerCooldownCapped(t *testing.T) {
	h := newHealthTracker()
	for range 20 {
		h.RecordFailure("brave")
	}

	h.mu.Lock()
	cooldown := h.backends["brave"].cooldown
	h.mu.Unlock()

	if cooldown > maxCooldown {
		t.Errorf("cooldown should be capped at %v, got %v", maxCooldown, cooldown)
	}
}

func TestHealthTrackerCooldownExpires(t *testing.T) {
	h := newHealthTracker()
	h.RecordFailure("brave")

	// Backdate the failure.
	h.mu.Lock()
	h.backends["brave"].lastFail = time.Now().Add(-3 * minCooldown)
	h.mu.Unlock()

	if !h.IsHealthy("brave") {
		t.Error("backend should be healthy after cooldown expires")
	}
}

func TestHealthTrackerIndependentBackends(t *testing.T) {
	h := newHealthTracker()
	h.RecordFailure("brave")

	if !h.IsHealthy("duckduckgo") {
		t.Error("failure in brave should not affect duckduckgo")
	}
}
