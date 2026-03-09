package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/priyanshujain/openbotkit/provider"
)

// Tool is a callable tool with a JSON schema.
type Tool interface {
	Name() string
	Description() string
	InputSchema() json.RawMessage
	Execute(ctx context.Context, input json.RawMessage) (string, error)
}

// Registry holds registered tools and implements agent.ToolExecutor.
type Registry struct {
	tools map[string]Tool
}

// NewRegistry creates an empty tool registry.
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

// Register adds a tool to the registry.
func (r *Registry) Register(t Tool) {
	r.tools[t.Name()] = t
}

const maxOutputBytes = 524288 // 512KB

// Execute implements agent.ToolExecutor.
func (r *Registry) Execute(ctx context.Context, call provider.ToolCall) (string, error) {
	t, ok := r.tools[call.Name]
	if !ok {
		return "", fmt.Errorf("unknown tool %q", call.Name)
	}
	output, err := t.Execute(ctx, call.Input)
	if len(output) > maxOutputBytes {
		output = output[:maxOutputBytes] + fmt.Sprintf(
			"\n...[output truncated, showing first 512KB of %dKB]", len(output)/1024)
	}
	return output, err
}

// ToolSchemas implements agent.ToolExecutor.
func (r *Registry) ToolSchemas() []provider.Tool {
	schemas := make([]provider.Tool, 0, len(r.tools))
	for _, t := range r.tools {
		schemas = append(schemas, provider.Tool{
			Name:        t.Name(),
			Description: t.Description(),
			InputSchema: t.InputSchema(),
		})
	}
	return schemas
}
