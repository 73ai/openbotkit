package agent

import (
	"context"
	"fmt"

	"github.com/priyanshujain/openbotkit/provider"
)

// ToolExecutor handles tool call execution.
type ToolExecutor interface {
	Execute(ctx context.Context, call provider.ToolCall) (string, error)
	ToolSchemas() []provider.Tool
}

// Agent orchestrates the conversation between user, LLM, and tools.
type Agent struct {
	provider    provider.Provider
	model       string
	system      string
	executor    ToolExecutor
	history     []provider.Message
	maxIter     int
	maxHistory  int
	rateLimiter *provider.RateLimiter
}

// Option configures an Agent.
type Option func(*Agent)

// WithSystem sets the system prompt.
func WithSystem(system string) Option {
	return func(a *Agent) { a.system = system }
}

// WithMaxIterations sets the maximum number of tool-use iterations.
func WithMaxIterations(n int) Option {
	return func(a *Agent) { a.maxIter = n }
}

// WithMaxHistory sets the maximum number of history messages before compaction.
func WithMaxHistory(n int) Option {
	return func(a *Agent) { a.maxHistory = n }
}

// WithRateLimit sets a rate limit on LLM API calls (requests per hour).
func WithRateLimit(requestsPerHour int) Option {
	return func(a *Agent) { a.rateLimiter = provider.NewRateLimiter(requestsPerHour) }
}

// New creates a new Agent.
func New(p provider.Provider, model string, executor ToolExecutor, opts ...Option) *Agent {
	a := &Agent{
		provider:   p,
		model:      model,
		executor:   executor,
		maxIter:    25,
		maxHistory: defaultMaxHistory,
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// Run sends a user message and returns the final assistant text response.
// It handles multi-turn tool use loops internally.
func (a *Agent) Run(ctx context.Context, input string) (string, error) {
	a.history = append(a.history, provider.NewTextMessage(provider.RoleUser, input))

	for i := range a.maxIter {
		a.compactHistory()
		if a.rateLimiter != nil {
			if err := a.rateLimiter.Wait(ctx); err != nil {
				return "", fmt.Errorf("rate limiter: %w", err)
			}
		}
		resp, err := a.provider.Chat(ctx, provider.ChatRequest{
			Model:     a.model,
			System:    a.system,
			Messages:  a.history,
			Tools:     a.executor.ToolSchemas(),
			MaxTokens: 4096,
		})
		if err != nil {
			return "", fmt.Errorf("chat (iteration %d): %w", i, err)
		}

		// Append assistant response to history.
		a.history = append(a.history, provider.Message{
			Role:    provider.RoleAssistant,
			Content: resp.Content,
		})

		if resp.StopReason != provider.StopToolUse {
			return resp.TextContent(), nil
		}

		// Execute tool calls and collect results.
		var results []provider.ContentBlock
		for _, call := range resp.ToolCalls() {
			output, err := a.executor.Execute(ctx, call)
			isError := err != nil
			content := ScrubCredentials(output)
			if isError {
				content = ScrubCredentials(err.Error())
			}
			results = append(results, provider.ContentBlock{
				Type: provider.ContentToolResult,
				ToolResult: &provider.ToolResult{
					ToolUseID: call.ID,
					Content:   content,
					IsError:   isError,
				},
			})
		}

		// Append tool results to history.
		a.history = append(a.history, provider.Message{
			Role:    provider.RoleUser,
			Content: results,
		})
	}

	return "", fmt.Errorf("max iterations (%d) reached", a.maxIter)
}
