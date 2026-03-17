package tools

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/73ai/openbotkit/internal/skills"
	"github.com/73ai/openbotkit/oauth/google"
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
				"description": "The gws command without --params or --json flags (e.g. 'drive files list', 'calendar events list')"
			},
			"params": {
				"type": "object",
				"description": "URL/query parameters as a JSON object (becomes --params flag)"
			},
			"body": {
				"type": "object",
				"description": "Request body as a JSON object (becomes --json flag)"
			}
		},
		"required": ["command"]
	}`)
}

type gwsInput struct {
	Command string          `json:"command"`
	Params  json.RawMessage `json:"params,omitempty"`
	Body    json.RawMessage `json:"body,omitempty"`
}

func (g *GWSExecuteTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var in gwsInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("parse input: %w", err)
	}
	if in.Command == "" {
		return "", fmt.Errorf("command is required")
	}

	slog.Info("gws_execute called", "command", in.Command)
	args := strings.Fields(in.Command)
	// Strip leading "gws" if present.
	if len(args) > 0 && args[0] == "gws" {
		args = args[1:]
	}
	// Append structured params/body as flags.
	if len(in.Params) > 0 && string(in.Params) != "null" {
		args = append(args, "--params", string(in.Params))
	}
	if len(in.Body) > 0 && string(in.Body) != "null" {
		args = append(args, "--json", string(in.Body))
	}
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
		slog.Warn("gws_execute: token error, attempting re-auth", "error", err)
		// Token expired or refresh failed — trigger re-consent and retry.
		service := gwsServiceFromCommand(args)
		scopes := g.scopesForService(service)
		if len(scopes) == 0 {
			return "", fmt.Errorf("get token: %w", err)
		}
		if cerr := g.requestConsent(ctx, scopes); cerr != nil {
			return "", fmt.Errorf("get token: %w (re-auth also failed: %v)", err, cerr)
		}
		env, err = g.bridge.Env(ctx)
		if err != nil {
			return "", fmt.Errorf("get token after re-auth: %w", err)
		}
	}
	runCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	out, runErr := g.runner.Run(runCtx, args, env)

	// If Google returns insufficient scopes, the local DB is stale.
	// Trigger progressive consent and retry once.
	if runErr != nil && strings.Contains(out, "ACCESS_TOKEN_SCOPE_INSUFFICIENT") {
		slog.Warn("gws_execute: scope insufficient, triggering re-auth")
		service := gwsServiceFromCommand(args)
		scopes := g.scopesForService(service)
		if len(scopes) > 0 {
			if cerr := g.requestConsent(ctx, scopes); cerr == nil {
				env, err = g.bridge.Env(ctx)
				if err == nil {
					retryCtx, retryCancel := context.WithTimeout(ctx, 30*time.Second)
					defer retryCancel()
					out, runErr = g.runner.Run(retryCtx, args, env)
				}
			}
		}
	}

	// If the API is disabled, flag it so the agent can extract the
	// enablement URL from the error and share it with the user.
	if runErr != nil && (strings.Contains(out, "SERVICE_DISABLED") || strings.Contains(out, "API has not been used")) {
		slog.Warn("gws_execute: API disabled for service", "service", gwsServiceFromCommand(args))
		out = out + "\n\nNote: The Google API for this service is not enabled. The error above contains an activation URL — share it with the user so they can enable the API."
	}

	out = TruncateHeadTail(out, MaxLinesHeadTail, MaxLinesHeadTail)
	out = TruncateBytes(out, MaxOutputBytes)
	return out, runErr
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

	// After first-time auth, discover the account from the token store.
	if g.account == "" {
		accounts, err := g.google.Accounts(ctx)
		if err == nil && len(accounts) > 0 {
			g.account = accounts[0]
			g.bridge.SetAccount(accounts[0])
		}
	}

	if err := g.interactor.Notify("Access granted, thanks!"); err != nil {
		return fmt.Errorf("notify: %w", err)
	}
	return nil
}

func (g *GWSExecuteTool) isWriteCommand(args []string) bool {
	if g.manifest == nil {
		return false
	}
	// Build the expected skill name from the command args.
	// e.g. ["calendar", "+agenda"] → "gws-calendar-agenda"
	service := gwsServiceFromCommand(args)
	for _, arg := range args {
		if sub, ok := strings.CutPrefix(arg, "+"); ok {
			skillName := "gws-" + service + "-" + sub
			if entry, ok := g.manifest.Skills[skillName]; ok {
				return entry.Write
			}
			return true // unknown + command, assume write to be safe
		}
	}
	return false
}

// serviceToScope maps gws short service names to full Google API scope URLs.
var serviceToScope = map[string]string{
	"calendar": "https://www.googleapis.com/auth/calendar",
	"drive":    "https://www.googleapis.com/auth/drive",
	"docs":     "https://www.googleapis.com/auth/documents",
	"sheets":   "https://www.googleapis.com/auth/spreadsheets",
	"tasks":    "https://www.googleapis.com/auth/tasks",
	"people":   "https://www.googleapis.com/auth/contacts",
	"gmail":    "https://www.googleapis.com/auth/gmail.modify",
}

func (g *GWSExecuteTool) scopesForService(service string) []string {
	// Check manifest for GWS skills first.
	if g.manifest != nil {
		for _, entry := range g.manifest.Skills {
			if entry.Source == "gws" && len(entry.Scopes) > 0 && entry.Scopes[0] == service {
				if scope, ok := serviceToScope[service]; ok {
					return []string{scope}
				}
			}
		}
	}
	// Fall back to the scope map directly so progressive consent works
	// even for services without a GWS-generated skill (e.g. gmail).
	if scope, ok := serviceToScope[service]; ok {
		return []string{scope}
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
