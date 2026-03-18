package memory

import (
	"context"
	"time"

	"github.com/73ai/openbotkit/provider"
)

// LLM is the interface used by Extract and Reconcile for LLM calls.
// Satisfied by RouterLLM adapter or any mock in tests.
type LLM interface {
	Chat(ctx context.Context, req provider.ChatRequest) (*provider.ChatResponse, error)
}

// RouterLLM adapts a provider.Router to the LLM interface using a fixed tier.
type RouterLLM struct {
	Router *provider.Router
	Tier   provider.ModelTier
}

func (r *RouterLLM) Chat(ctx context.Context, req provider.ChatRequest) (*provider.ChatResponse, error) {
	return r.Router.Chat(ctx, r.Tier, req)
}

type Category string

const (
	CategoryIdentity     Category = "identity"
	CategoryPreference   Category = "preference"
	CategoryRelationship Category = "relationship"
	CategoryProject      Category = "project"
)

type Memory struct {
	ID        int64
	Content   string
	Category  Category
	Source    string // "history", "whatsapp", "gmail", "applenotes", "manual"
	SourceRef string // optional reference (session_id, etc.)
	CreatedAt time.Time
	UpdatedAt time.Time
}
