package provider

import (
	"context"
	"fmt"

	"github.com/73ai/openbotkit/config"
)

// ModelTier represents the complexity level for model routing.
type ModelTier string

const (
	TierDefault ModelTier = "default"
	TierComplex ModelTier = "complex"
	TierFast    ModelTier = "fast"
	TierNano    ModelTier = "nano"
)

// Router selects the appropriate provider and model for a given tier.
type Router struct {
	registry *Registry
	models   *config.ModelsConfig
}

// NewRouter creates a new model router.
func NewRouter(registry *Registry, models *config.ModelsConfig) *Router {
	return &Router{registry: registry, models: models}
}

// Chat routes a request to the appropriate provider and model based on tier.
func (r *Router) Chat(ctx context.Context, tier ModelTier, req ChatRequest) (*ChatResponse, error) {
	p, model, err := r.resolve(tier)
	if err != nil {
		return nil, err
	}
	req.Model = model
	return p.Chat(ctx, req)
}

// StreamChat routes a streaming request to the appropriate provider.
func (r *Router) StreamChat(ctx context.Context, tier ModelTier, req ChatRequest) (<-chan StreamEvent, error) {
	p, model, err := r.resolve(tier)
	if err != nil {
		return nil, err
	}
	req.Model = model
	return p.StreamChat(ctx, req)
}

// resolve returns the provider and model for the given tier.
// Cascade order: nano → fast → default, complex → default.
func (r *Router) resolve(tier ModelTier) (Provider, string, error) {
	spec := r.specForTier(tier)
	if spec == "" && tier == TierNano {
		spec = r.specForTier(TierFast)
	}
	if spec == "" {
		spec = r.models.Default
	}
	if spec == "" {
		return nil, "", fmt.Errorf("no model configured for tier %q", tier)
	}

	providerName, model, err := ParseModelSpec(spec)
	if err != nil {
		return nil, "", err
	}

	p, ok := r.registry.Get(providerName)
	if !ok {
		return nil, "", fmt.Errorf("provider %q not found in registry", providerName)
	}

	return p, model, nil
}

func (r *Router) specForTier(tier ModelTier) string {
	switch tier {
	case TierComplex:
		return r.models.Complex
	case TierFast:
		return r.models.Fast
	case TierNano:
		return r.models.Nano
	default:
		return r.models.Default
	}
}
