package config

import "slices"

// ModelInfo describes a model available for profile configuration.
type ModelInfo struct {
	Provider       string
	ID             string
	Label          string
	ContextWindow  int
	RecommendedFor []string // "default", "complex", "fast", "nano"
}

// ModelCatalog lists all models available for custom profile building.
var ModelCatalog = []ModelInfo{
	// Anthropic
	{Provider: "anthropic", ID: "claude-sonnet-4-6", Label: "Claude Sonnet 4.6 (balanced)", ContextWindow: 200000, RecommendedFor: []string{"default", "complex"}},
	{Provider: "anthropic", ID: "claude-opus-4-6", Label: "Claude Opus 4.6 (most capable)", ContextWindow: 200000, RecommendedFor: []string{"complex"}},
	{Provider: "anthropic", ID: "claude-haiku-4-5", Label: "Claude Haiku 4.5 (fast, cheap)", ContextWindow: 200000, RecommendedFor: []string{"default", "fast", "nano"}},

	// OpenAI
	{Provider: "openai", ID: "gpt-4o", Label: "GPT-4o (most capable)", ContextWindow: 128000, RecommendedFor: []string{"default", "complex"}},
	{Provider: "openai", ID: "gpt-4o-mini", Label: "GPT-4o Mini (fast, cheap)", ContextWindow: 128000, RecommendedFor: []string{"default", "fast", "nano"}},

	// Gemini
	{Provider: "gemini", ID: "gemini-2.5-pro", Label: "Gemini 2.5 Pro (most capable)", ContextWindow: 1048576, RecommendedFor: []string{"complex"}},
	{Provider: "gemini", ID: "gemini-2.5-flash", Label: "Gemini 2.5 Flash (balanced)", ContextWindow: 1048576, RecommendedFor: []string{"default", "complex"}},
	{Provider: "gemini", ID: "gemini-2.0-flash-lite", Label: "Gemini 2.0 Flash Lite (fastest)", ContextWindow: 1048576, RecommendedFor: []string{"fast", "nano"}},

	// OpenRouter
	{Provider: "openrouter", ID: "anthropic/claude-sonnet-4-6", Label: "Claude Sonnet 4.6 via OpenRouter", ContextWindow: 200000, RecommendedFor: []string{"default", "complex"}},
	{Provider: "openrouter", ID: "anthropic/claude-haiku-4-5", Label: "Claude Haiku 4.5 via OpenRouter", ContextWindow: 200000, RecommendedFor: []string{"default", "fast"}},
	{Provider: "openrouter", ID: "anthropic/claude-opus-4-6", Label: "Claude Opus 4.6 via OpenRouter", ContextWindow: 200000, RecommendedFor: []string{"complex"}},
	{Provider: "openrouter", ID: "google/gemini-2.0-flash-lite", Label: "Gemini Flash Lite via OpenRouter", ContextWindow: 1048576, RecommendedFor: []string{"fast", "nano"}},
	{Provider: "openrouter", ID: "mistralai/mistral-medium-3.1", Label: "Mistral Medium 3.1 via OpenRouter", ContextWindow: 131072, RecommendedFor: []string{"default"}},

	// Groq
	{Provider: "groq", ID: "llama-3.1-8b-instant", Label: "Llama 3.1 8B (fastest)", ContextWindow: 131072, RecommendedFor: []string{"fast", "nano"}},
	{Provider: "groq", ID: "llama-3.3-70b-versatile", Label: "Llama 3.3 70B (versatile)", ContextWindow: 131072, RecommendedFor: []string{"default"}},
	{Provider: "groq", ID: "llama-4-scout-17b-16e", Label: "Llama 4 Scout 17B", ContextWindow: 131072, RecommendedFor: []string{"default", "complex"}},
}

// ModelsForProviders returns models matching the given provider names.
func ModelsForProviders(providers []string) []ModelInfo {
	provSet := make(map[string]bool, len(providers))
	for _, p := range providers {
		provSet[p] = true
	}
	var result []ModelInfo
	for _, m := range ModelCatalog {
		if provSet[m.Provider] {
			result = append(result, m)
		}
	}
	return result
}

// ModelsForTier filters models to those recommended for a given tier.
func ModelsForTier(models []ModelInfo, tier string) []ModelInfo {
	var result []ModelInfo
	for _, m := range models {
		if slices.Contains(m.RecommendedFor, tier) {
			result = append(result, m)
		}
	}
	return result
}
