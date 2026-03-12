package tools

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/priyanshujain/openbotkit/internal/skills"
	"github.com/priyanshujain/openbotkit/oauth/google"
)

const defaultAuthTimeout = 5 * time.Minute

// GWSToolConfig configures a GWSExecuteTool.
type GWSToolConfig struct {
	Interactor    Interactor
	ScopeChecker  ScopeChecker
	Bridge        *TokenBridge
	ScopeWaiter   *google.ScopeWaiter
	Google        *google.Google
	Account       string
	Manifest      *skills.Manifest
	Runner        CommandRunner
	AuthTimeout   time.Duration
	ApprovalRules *ApprovalRuleSet
}

// GWSExecuteTool routes all gws commands through a single tool
// with scope checking, progressive consent, and write approval.
type GWSExecuteTool struct {
	interactor    Interactor
	scopeChecker  ScopeChecker
	bridge        *TokenBridge
	scopeWaiter   *google.ScopeWaiter
	google        *google.Google
	account       string
	manifest      *skills.Manifest
	runner        CommandRunner
	authTimeout   time.Duration
	approvalRules *ApprovalRuleSet
}

func NewGWSExecuteTool(cfg GWSToolConfig) *GWSExecuteTool {
	timeout := cfg.AuthTimeout
	if timeout == 0 {
		timeout = defaultAuthTimeout
	}
	return &GWSExecuteTool{
		interactor:    cfg.Interactor,
		scopeChecker:  cfg.ScopeChecker,
		bridge:        cfg.Bridge,
		scopeWaiter:   cfg.ScopeWaiter,
		google:        cfg.Google,
		account:       cfg.Account,
		manifest:      cfg.Manifest,
		runner:        cfg.Runner,
		authTimeout:   timeout,
		approvalRules: cfg.ApprovalRules,
	}
}

func (g *GWSExecuteTool) Name() string        { return "gws_execute" }
func (g *GWSExecuteTool) Description() string { return "Execute a Google Workspace CLI (gws) command" }
func (g *GWSExecuteTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"command": {
				"type": "string",
				"description": "The gws command to execute (e.g. 'calendar events.list --maxResults 10')"
			}
		},
		"required": ["command"]
	}`)
}

type gwsInput struct {
	Command string `json:"command"`
}

func (g *GWSExecuteTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var in gwsInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("parse input: %w", err)
	}
	if in.Command == "" {
		return "", fmt.Errorf("command is required")
	}

	args := strings.Fields(in.Command)
	service := gwsServiceFromCommand(args)
	isWrite := g.isWriteCommand(args)

	// Scope check + progressive consent.
	if service != "" {
		requiredScopes := g.scopesForService(service)
		if len(requiredScopes) > 0 {
			has, err := g.scopeChecker.HasScopes(g.account, requiredScopes)
			if err != nil {
				return "", fmt.Errorf("check scopes: %w", err)
			}
			if !has {
				if err := g.requestConsent(ctx, requiredScopes); err != nil {
					return "", err
				}
			}
		}
	}

	// Write approval.
	if isWrite {
		var opts []GuardOption
		if g.approvalRules != nil {
			opts = append(opts, WithApprovalRules(g.approvalRules, "gws_execute", input))
		}
		return GuardedAction(ctx, g.interactor, RiskHigh, fmt.Sprintf("Run gws command: %s", in.Command), func() (string, error) {
			return g.run(ctx, args)
		}, opts...)
	}

	return g.run(ctx, args)
}

func (g *GWSExecuteTool) run(ctx context.Context, args []string) (string, error) {
	env, err := g.bridge.Env(ctx)
	if err != nil {
		return "", fmt.Errorf("get token: %w", err)
	}
	return g.runner.Run(ctx, args, env)
}

func (g *GWSExecuteTool) requestConsent(ctx context.Context, scopes []string) error {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Errorf("generate state: %w", err)
	}
	state := "gws-" + hex.EncodeToString(b[:])
	url, err := g.google.AuthURL(g.account, scopes, state)
	if err != nil {
		return fmt.Errorf("generate auth URL: %w", err)
	}
	if err := g.interactor.Notify("I need additional Google access to complete this request."); err != nil {
		return fmt.Errorf("notify: %w", err)
	}
	if err := g.interactor.NotifyLink("Tap to grant access", url); err != nil {
		return fmt.Errorf("notify link: %w", err)
	}

	if err := g.scopeWaiter.Wait(state, g.authTimeout, scopes, g.account); err != nil {
		return fmt.Errorf("auth: %w", err)
	}
	if err := g.interactor.Notify("Access granted, thanks!"); err != nil {
		return fmt.Errorf("notify: %w", err)
	}
	return nil
}

func (g *GWSExecuteTool) isWriteCommand(args []string) bool {
	for _, arg := range args {
		if strings.HasPrefix(arg, "+") {
			return true
		}
	}
	return false
}

func (g *GWSExecuteTool) scopesForService(service string) []string {
	if g.manifest == nil {
		return nil
	}
	for _, entry := range g.manifest.Skills {
		if entry.Source == "gws" && len(entry.Scopes) > 0 && entry.Scopes[0] == service {
			return entry.Scopes
		}
	}
	return nil
}

// gwsServiceFromCommand extracts the service name from gws command args.
func gwsServiceFromCommand(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}
