package provider

import "context"

// Provider is the universal LLM provider interface.
// All model providers (Anthropic, OpenAI, Gemini, etc.) implement this.
type Provider interface {
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	StreamChat(ctx context.Context, req ChatRequest) (<-chan StreamEvent, error)
}
