package tools

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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
	Tracker    *TaskTracker  // nil = sync-only (Phase 1 behavior)
}

const progressThrottle = 30 * time.Second

// DelegateTaskTool delegates complex tasks to external AI CLI agents.
type DelegateTaskTool struct {
	interactor    Interactor
	agents        []AgentInfo
	timeout       time.Duration
	runners       map[AgentKind]AgentRunnerInterface
	streamRunners map[AgentKind]StreamRunnerInterface
	tracker       *TaskTracker
}

// NewDelegateTaskTool creates a new delegate_task tool.
func NewDelegateTaskTool(cfg DelegateTaskConfig) *DelegateTaskTool {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = defaultDelegateTimeout
	}
	runners := make(map[AgentKind]AgentRunnerInterface, len(cfg.Agents))
	sRunners := make(map[AgentKind]StreamRunnerInterface, len(cfg.Agents))
	for _, a := range cfg.Agents {
		runners[a.Kind] = NewAgentRunner(a)
		sRunners[a.Kind] = NewStreamRunner(a)
	}
	return &DelegateTaskTool{
		interactor:    cfg.Interactor,
		agents:        cfg.Agents,
		timeout:       timeout,
		runners:       runners,
		streamRunners: sRunners,
		tracker:       cfg.Tracker,
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
		"steps": {
			"type": "array",
			"items": {"type": "string"},
			"description": "Optional ordered steps for the agent to follow"
		},
		"output_format": {
			"type": "string",
			"description": "Desired output format (e.g. markdown, json, plain)"
		},
		"max_budget_usd": {
			"type": "number",
			"description": "Maximum API cost budget in USD (Claude only)"
		},
		"agent": {
			"type": "string",
			"enum": %s,
			"description": "Which agent to use (auto-selects best available if omitted)"
		},
		"async": {
			"type": "boolean",
			"description": "Run in background and return immediately with a task_id (default false)"
		}
	},
	"required": ["task"]
}`, agentEnum))
}

type delegateTaskInput struct {
	Task         string   `json:"task"`
	Steps        []string `json:"steps,omitempty"`
	OutputFormat string   `json:"output_format,omitempty"`
	MaxBudgetUSD float64  `json:"max_budget_usd,omitempty"`
	Agent        string   `json:"agent,omitempty"`
	Async        bool     `json:"async,omitempty"`
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

	prompt := buildPrompt(in.Task, in.Steps, in.OutputFormat)
	preview := truncateUTF8(in.Task, 80)
	desc := fmt.Sprintf("Delegate to %s: %s", kind, preview)

	var runOpts []RunOption
	if in.MaxBudgetUSD > 0 {
		runOpts = append(runOpts, WithMaxBudget(in.MaxBudgetUSD))
	}

	// Async mode: if tracker is set and async requested, run in background.
	if in.Async && d.tracker != nil {
		return d.executeAsync(ctx, runner, kind, prompt, preview, desc, runOpts)
	}

	return GuardedWrite(ctx, d.interactor, desc, func() (string, error) {
		return runner.Run(ctx, prompt, d.timeout, runOpts...)
	})
}

// buildPrompt combines task, steps, and output format into a single prompt.
func buildPrompt(task string, steps []string, outputFormat string) string {
	if len(steps) == 0 && outputFormat == "" {
		return task
	}
	var b strings.Builder
	b.WriteString(task)
	if len(steps) > 0 {
		b.WriteString("\n\nSteps:\n")
		for i, s := range steps {
			fmt.Fprintf(&b, "%d. %s\n", i+1, s)
		}
	}
	if outputFormat != "" {
		fmt.Fprintf(&b, "\nOutput format: %s\n", outputFormat)
	}
	return b.String()
}

func (d *DelegateTaskTool) executeAsync(
	ctx context.Context,
	runner AgentRunnerInterface,
	kind AgentKind,
	task, preview, desc string,
	runOpts []RunOption,
) (string, error) {
	taskID, err := generateTaskID()
	if err != nil {
		return "", fmt.Errorf("generate task ID: %w", err)
	}

	if err := d.tracker.Start(taskID, task, kind); err != nil {
		return "", err
	}

	// Get approval before launching background goroutine.
	approved, err := d.interactor.RequestApproval(desc)
	if err != nil {
		d.tracker.Fail(taskID, err.Error())
		return "", fmt.Errorf("approval: %w", err)
	}
	if !approved {
		d.tracker.Fail(taskID, "denied by user")
		if nerr := d.interactor.Notify("Action not performed."); nerr != nil {
			return "", fmt.Errorf("notify denial: %w", nerr)
		}
		return "denied_by_user", nil
	}

	go d.runAsync(runner, kind, task, preview, taskID, runOpts)

	resp := struct {
		TaskID string `json:"task_id"`
		Status string `json:"status"`
	}{TaskID: taskID, Status: "running"}
	data, _ := json.Marshal(resp)
	return string(data), nil
}

func (d *DelegateTaskTool) runAsync(
	runner AgentRunnerInterface,
	kind AgentKind,
	task, preview, taskID string,
	runOpts []RunOption,
) {
	ctx, cancel := context.WithTimeout(context.Background(), d.timeout)
	defer cancel()

	// Try streaming runner for progress reporting.
	sr, ok := d.streamRunners[kind]
	if ok && sr != nil {
		var lastNotify time.Time
		onEvent := func(evt StreamEvent) {
			if time.Since(lastNotify) > progressThrottle && (evt.Type == "result" || evt.Type == "text") {
				d.interactor.Notify(fmt.Sprintf("Task %s progress: %s", taskID[:8], truncateUTF8(evt.Content, 200)))
				lastNotify = time.Now()
			}
		}
		output, err := sr.RunStream(ctx, task, d.timeout, onEvent, runOpts...)
		if err != nil {
			d.tracker.Fail(taskID, err.Error())
			d.interactor.Notify(fmt.Sprintf("Task failed: %s. %s", preview, err))
			return
		}
		d.tracker.Complete(taskID, output)
		d.interactor.Notify(fmt.Sprintf("Task completed: %s. Use check_task to see results.", preview))
		return
	}

	// Fallback to non-streaming runner.
	output, err := runner.Run(ctx, task, d.timeout, runOpts...)
	if err != nil {
		d.tracker.Fail(taskID, err.Error())
		d.interactor.Notify(fmt.Sprintf("Task failed: %s. %s", preview, err))
		return
	}
	d.tracker.Complete(taskID, output)
	d.interactor.Notify(fmt.Sprintf("Task completed: %s. Use check_task to see results.", preview))
}

func generateTaskID() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
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
