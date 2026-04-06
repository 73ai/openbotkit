package channel

import (
	"fmt"
	"sync"
)

// Registry holds pre-configured pushers keyed by channel name.
// Components say "send to telegram" without knowing credentials.
type Registry struct {
	mu      sync.RWMutex
	pushers map[string]Pusher
}

func NewRegistry() *Registry {
	return &Registry{pushers: make(map[string]Pusher)}
}

func (r *Registry) Register(name string, p Pusher) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pushers[name] = p
}

func (r *Registry) Get(name string) (Pusher, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.pushers[name]
	if !ok {
		return nil, fmt.Errorf("channel %q not registered", name)
	}
	return p, nil
}
