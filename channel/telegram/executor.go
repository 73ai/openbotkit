package telegram

import (
	"context"

	"github.com/73ai/openbotkit/agent"
	"github.com/73ai/openbotkit/provider"
)

// notifyingExecutor wraps a ToolExecutor and calls onToolStart before each execution.
type notifyingExecutor struct {
	delegate    agent.ToolExecutor
	onToolStart func(toolName string)
}

func (n *notifyingExecutor) Execute(ctx context.Context, call provider.ToolCall) (string, error) {
	if n.onToolStart != nil {
		n.onToolStart(call.Name)
	}
	return n.delegate.Execute(ctx, call)
}

func (n *notifyingExecutor) ToolSchemas() []provider.Tool {
	return n.delegate.ToolSchemas()
}
