package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/73ai/openbotkit/agent"
	"github.com/73ai/openbotkit/provider"
)

const defaultChildMaxIter = 10

// SubagentTool delegates a task to a child agent that runs synchronously.
type SubagentTool struct {
	provider    provider.Provider
	model       string
	toolFactory func() *Registry
	system      string
	maxIter     int
}

// SubagentConfig configures a SubagentTool.
type SubagentConfig struct {
	Provider    provider.Provider
	Model       string
	ToolFactory func() *Registry
	System      string
	MaxIter     int // 0 defaults to 10
}

func NewSubagentTool(cfg SubagentConfig) *SubagentTool {
	maxIter := cfg.MaxIter
	if maxIter == 0 {
		maxIter = defaultChildMaxIter
	}
	return &SubagentTool{
		provider:    cfg.Provider,
		model:       cfg.Model,
		toolFactory: cfg.ToolFactory,
		system:      cfg.System,
		maxIter:     maxIter,
	}
}

func (s *SubagentTool) Name() string { return "subagent" }
func (s *SubagentTool) Description() string {
	return "Delegate a self-contained task to a sub-agent that runs independently with its own context"
}
func (s *SubagentTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"task": {
				"type": "string",
				"description": "The task to delegate to the sub-agent"
			}
		},
		"required": ["task"]
	}`)
}

type subagentInput struct {
	Task string `json:"task"`
}

func (s *SubagentTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var in subagentInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("parse input: %w", err)
	}
	if in.Task == "" {
		return "", fmt.Errorf("task is required")
	}

	childReg := s.toolFactory()
	child := agent.New(
		s.provider, s.model, childReg,
		agent.WithSystem(s.system),
		agent.WithMaxIterations(s.maxIter),
	)
	out, err := child.Run(ctx, in.Task)
	out = TruncateHeadTail(out, MaxLinesHeadTail, MaxLinesHeadTail)
	out = TruncateBytes(out, MaxOutputBytes)
	return out, err
}
