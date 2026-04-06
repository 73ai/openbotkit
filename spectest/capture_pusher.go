package spectest

import (
	"context"
	"sync"
)

// CapturePusher records all pushed messages for test assertions.
type CapturePusher struct {
	mu       sync.Mutex
	messages []string
}

func (p *CapturePusher) Push(_ context.Context, message string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.messages = append(p.messages, message)
	return nil
}

func (p *CapturePusher) Messages() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	cp := make([]string, len(p.messages))
	copy(cp, p.messages)
	return cp
}
