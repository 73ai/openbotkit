package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const defaultDelegateTimeout = 5 * time.Minute

// DelegateTaskConfig configures the delegate_task tool.
type DelegateTaskConfig struct {
	Interactor Interactor
	Agents     []AgentInfo
	Timeout    time.Duration // default 5m
}

// DelegateTaskTool delegates complex tasks to external AI CLI agents.
type DelegateTaskTool struct {
	interactor Interactor
	agents     []AgentInfo
	timeout    time.Duration
	runners    map[AgentKind]AgentRunnerInterface
}

// NewDelegateTaskTool creates a new delegate_task tool.
func NewDelegateTaskTool(cfg DelegateTaskConfig) *DelegateTaskTool {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = defaultDelegateTimeout
	}
	runners := make(map[AgentKind]AgentRunnerInterface, len(cfg.Agents))
	for _, a := range cfg.Agents {
		runners[a.Kind] = NewAgentRunner(a)
	}
	return &DelegateTaskTool{
		interactor: cfg.Interactor,
		agents:     cfg.Agents,
		timeout:    timeout,
		runners:    runners,
	}
}

func (d *DelegateTaskTool) Name() string { return "delegate_task" }

func (d *DelegateTaskTool) Description() string {
	return "Delegate a complex task to an external AI agent (requires user approval)"
}

func (d *DelegateTaskTool) InputSchema() json.RawMessage {
	agentEnum := d.agentEnumJSON()
	return json.RawMessage(fmt.Sprintf(`{
	"type": "object",
	"properties": {
		"task": {
			"type": "string",
			"description": "The task to delegate to the external agent"
		},
		"agent": {
			"type": "string",
			"enum": %s,
			"description": "Which agent to use (auto-selects best available if omitted)"
		}
	},
	"required": ["task"]
}`, agentEnum))
}

type delegateTaskInput struct {
	Task  string `json:"task"`
	Agent string `json:"agent,omitempty"`
}

func (d *DelegateTaskTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var in delegateTaskInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("parse input: %w", err)
	}
	if strings.TrimSpace(in.Task) == "" {
		return "", fmt.Errorf("task is required")
	}

	runner, kind, err := d.selectRunner(in.Agent)
	if err != nil {
		return "", err
	}

	preview := truncateUTF8(in.Task, 80)
	desc := fmt.Sprintf("Delegate to %s: %s", kind, preview)

	return GuardedWrite(ctx, d.interactor, desc, func() (string, error) {
		return runner.Run(ctx, in.Task, d.timeout)
	})
}

func (d *DelegateTaskTool) selectRunner(agent string) (AgentRunnerInterface, AgentKind, error) {
	if agent == "" {
		if len(d.agents) == 0 {
			return nil, "", fmt.Errorf("no agents available")
		}
		kind := d.agents[0].Kind
		return d.runners[kind], kind, nil
	}
	kind := AgentKind(agent)
	runner, ok := d.runners[kind]
	if !ok {
		return nil, "", fmt.Errorf("agent %q not available", agent)
	}
	return runner, kind, nil
}

func (d *DelegateTaskTool) agentEnumJSON() string {
	names := make([]string, len(d.agents))
	for i, a := range d.agents {
		names[i] = fmt.Sprintf("%q", a.Kind)
	}
	return "[" + strings.Join(names, ", ") + "]"
}
