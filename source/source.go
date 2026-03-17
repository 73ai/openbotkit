package source

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/73ai/openbotkit/store"
)

type Status struct {
	Connected    bool
	Accounts     []string
	ItemCount    int64
	LastSyncedAt *time.Time
}

type Source interface {
	Name() string
	Status(ctx context.Context, db *store.DB) (*Status, error)
}

var (
	registryMu sync.RWMutex
	registry   = map[string]Source{}
)

func Register(s Source) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[s.Name()] = s
}

func Get(name string) (Source, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	s, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("source %q not registered", name)
	}
	return s, nil
}

func All() []Source {
	registryMu.RLock()
	defer registryMu.RUnlock()
	sources := make([]Source, 0, len(registry))
	for _, s := range registry {
		sources = append(sources, s)
	}
	return sources
}
