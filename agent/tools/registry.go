package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

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

// NewStandardRegistry creates a registry with the standard tool set
// (bash, file_read, file_write, file_edit, load_skills, search_skills).
func NewStandardRegistry() *Registry {
	r := NewRegistry()
	r.Register(NewBashTool(30 * time.Second))
	r.Register(&FileReadTool{})
	r.Register(&FileWriteTool{})
	r.Register(&FileEditTool{})
	r.Register(&LoadSkillsTool{})
	r.Register(&SearchSkillsTool{})
	return r
}

// Has returns true if a tool with the given name is registered.
func (r *Registry) Has(name string) bool {
	_, ok := r.tools[name]
	return ok
}

// ToolNames returns sorted tool names registered in the registry.
func (r *Registry) ToolNames() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
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
