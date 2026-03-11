package websearch

import (
	"sync"
	"time"
)

const (
	minCooldown = 30 * time.Second
	maxCooldown = 5 * time.Minute
)

type failureInfo struct {
	count    int
	lastFail time.Time
	cooldown time.Duration
}

type healthTracker struct {
	mu       sync.Mutex
	backends map[string]*failureInfo
}

func newHealthTracker() *healthTracker {
	return &healthTracker{backends: make(map[string]*failureInfo)}
}

func (h *healthTracker) IsHealthy(name string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	info, ok := h.backends[name]
	if !ok {
		return true
	}
	return time.Since(info.lastFail) > info.cooldown
}

func (h *healthTracker) RecordFailure(name string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	info, ok := h.backends[name]
	if !ok {
		info = &failureInfo{cooldown: minCooldown}
		h.backends[name] = info
	}

	info.count++
	info.lastFail = time.Now()
	info.cooldown *= 2
	if info.cooldown < minCooldown {
		info.cooldown = minCooldown
	}
	if info.cooldown > maxCooldown {
		info.cooldown = maxCooldown
	}
}

func (h *healthTracker) RecordSuccess(name string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.backends, name)
}
