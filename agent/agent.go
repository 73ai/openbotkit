package agent

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/73ai/openbotkit/provider"
)

// ToolExecutor handles tool call execution.
type ToolExecutor interface {
	Execute(ctx context.Context, call provider.ToolCall) (string, error)
	ToolSchemas() []provider.Tool
}

// UsageRecorder records per-call token usage.
type UsageRecorder interface {
	RecordUsage(model string, usage provider.Usage)
}

// Summarizer generates a condensed summary of conversation messages.
type Summarizer interface {
	Summarize(ctx context.Context, messages []provider.Message) (string, error)
}

// Agent orchestrates the conversation between user, LLM, and tools.
type Agent struct {
	provider            provider.Provider
	model               string
	system              string
	systemBlocks        []provider.SystemBlock
	executor            ToolExecutor
	history             []provider.Message
	maxIter             int
	maxHistory          int
	contextWindow       int
	compactionThreshold float64
	lastInputTokens     int
	summarizer          Summarizer
	rateLimiter         *provider.RateLimiter
	usageRecorder       UsageRecorder
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

// WithSystemBlocks sets structured system prompt blocks with cache control.
func WithSystemBlocks(blocks []provider.SystemBlock) Option {
	return func(a *Agent) { a.systemBlocks = blocks }
}

// WithUsageRecorder sets a recorder for per-call token usage.
func WithUsageRecorder(r UsageRecorder) Option {
	return func(a *Agent) { a.usageRecorder = r }
}

// WithHistory seeds the agent with prior conversation messages.
func WithHistory(msgs []provider.Message) Option {
	return func(a *Agent) { a.history = msgs }
}

// WithContextWindow sets the model context window size in tokens.
func WithContextWindow(tokens int) Option {
	return func(a *Agent) { a.contextWindow = tokens }
}

// WithCompactionThreshold sets the context fill percentage that triggers compaction.
func WithCompactionThreshold(pct float64) Option {
	return func(a *Agent) { a.compactionThreshold = pct }
}

// WithSummarizer sets the summarizer for LLM-based compaction.
func WithSummarizer(s Summarizer) Option {
	return func(a *Agent) { a.summarizer = s }
}

// New creates a new Agent.
func New(p provider.Provider, model string, executor ToolExecutor, opts ...Option) *Agent {
	a := &Agent{
		provider:            p,
		model:               model,
		executor:            executor,
		maxIter:             25,
		maxHistory:          defaultMaxHistory,
		contextWindow:       1_000_000,
		compactionThreshold: 0.6,
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
		a.microcompact()
		a.compactHistory(ctx)
		if a.rateLimiter != nil {
			if err := a.rateLimiter.Wait(ctx); err != nil {
				return "", fmt.Errorf("rate limiter: %w", err)
			}
		}
		slog.Info("llm request", "iteration", i, "messages", len(a.history), "tools", len(a.executor.ToolSchemas()))
		resp, err := a.provider.Chat(ctx, provider.ChatRequest{
			Model:        a.model,
			System:       a.system,
			SystemBlocks: a.systemBlocks,
			Messages:     a.history,
			Tools:        a.executor.ToolSchemas(),
			MaxTokens:    4096,
		})
		if err != nil {
			return "", fmt.Errorf("chat (iteration %d): %w", i, err)
		}

		a.lastInputTokens = resp.Usage.InputTokens
		slog.Info("llm response", "iteration", i, "input_tokens", resp.Usage.InputTokens, "output_tokens", resp.Usage.OutputTokens, "stop", resp.StopReason)
		if a.usageRecorder != nil {
			a.usageRecorder.RecordUsage(a.model, resp.Usage)
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
				// Include both the output (e.g. stderr) and the error
				// so the LLM has enough context to retry intelligently.
				errMsg := ScrubCredentials(err.Error())
				if content != "" {
					content = content + "\n\nError: " + errMsg
				} else {
					content = errMsg
				}
			}
			results = append(results, provider.ContentBlock{
				Type: provider.ContentToolResult,
				ToolResult: &provider.ToolResult{
					ToolUseID: call.ID,
					Name:      call.Name,
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
