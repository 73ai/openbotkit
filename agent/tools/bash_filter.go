package tools

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// FilterResult indicates the outcome of a command filter check.
type FilterResult int

const (
	FilterAllow  FilterResult = iota // on allowlist, run freely
	FilterDeny                       // hard blocked
	FilterPrompt                     // not on allowlist, ask user
)

// CommandFilter validates shell commands against an allowlist or blocklist.
type CommandFilter struct {
	allowed   []string // if set, only these prefixes pass
	blocked   []string // if set, these prefixes are rejected
	softAllow bool     // if true, non-matching returns FilterPrompt instead of FilterDeny
}

// NewAllowlistFilter creates a filter that only permits commands
// whose first token matches one of the given prefixes.
func NewAllowlistFilter(prefixes []string) *CommandFilter {
	return &CommandFilter{allowed: prefixes}
}

// NewSoftAllowlistFilter creates a filter that auto-allows commands on the
// allowlist and returns FilterPrompt (not FilterDeny) for everything else.
// Use this for interactive mode where unknown commands should be approved by the user.
func NewSoftAllowlistFilter(prefixes []string) *CommandFilter {
	return &CommandFilter{allowed: prefixes, softAllow: true}
}

// NewBlocklistFilter creates a filter that rejects commands
// whose first token matches any of the given prefixes.
func NewBlocklistFilter(prefixes []string) *CommandFilter {
	return &CommandFilter{blocked: prefixes}
}

// CheckWithResult validates the given command string and returns a FilterResult
// indicating whether to allow, deny, or prompt the user.
func (f *CommandFilter) CheckWithResult(command string) (FilterResult, error) {
	if f == nil {
		return FilterAllow, nil
	}

	segments := splitShellSegments(command)
	for _, seg := range segments {
		result, err := f.checkSegmentResult(seg)
		if err != nil || result != FilterAllow {
			return result, err
		}
	}

	for _, sub := range extractSubstitutions(command) {
		result, err := f.CheckWithResult(sub)
		if err != nil {
			return result, fmt.Errorf("in command substitution: %w", err)
		}
		if result != FilterAllow {
			return result, nil
		}
	}

	return FilterAllow, nil
}

// Check validates the given command string. It splits on shell
// operators (|, &&, ;, ||) and checks each segment. It also
// detects command substitution via $() and backticks.
func (f *CommandFilter) Check(command string) error {
	result, err := f.CheckWithResult(command)
	if err != nil {
		return err
	}
	if result == FilterDeny {
		return fmt.Errorf("command not permitted")
	}
	if result == FilterPrompt {
		return fmt.Errorf("command requires approval")
	}
	return nil
}

// basename strips directory components so "/usr/bin/curl" → "curl".
func basename(token string) string {
	return filepath.Base(token)
}

// checkSegmentResult validates a single command segment and returns a FilterResult.
func (f *CommandFilter) checkSegmentResult(seg string) (FilterResult, error) {
	fields := strings.Fields(strings.TrimSpace(seg))
	if len(fields) == 0 {
		return FilterAllow, nil
	}
	if len(f.allowed) > 0 {
		return f.checkTokenResult(fields[0])
	}
	for _, tok := range fields {
		result, err := f.checkTokenResult(tok)
		if err != nil || result != FilterAllow {
			return result, err
		}
	}
	return FilterAllow, nil
}

func (f *CommandFilter) checkTokenResult(token string) (FilterResult, error) {
	base := basename(token)
	if len(f.allowed) > 0 {
		for _, prefix := range f.allowed {
			if base == prefix {
				return FilterAllow, nil
			}
		}
		if f.softAllow {
			return FilterPrompt, nil
		}
		return FilterDeny, fmt.Errorf("command %q not in allowlist", token)
	}
	for _, prefix := range f.blocked {
		if base == prefix {
			return FilterDeny, fmt.Errorf("command %q is blocked", token)
		}
	}
	return FilterAllow, nil
}

// splitShellSegments splits a command on |, &&, ;, and || operators.
// This is a simplified parser that handles common cases.
func splitShellSegments(cmd string) []string {
	// Replace operators with a sentinel, then split.
	// Order matters: || before | to avoid partial matches.
	s := cmd
	for _, op := range []string{"||", "&&", "|", ";"} {
		s = strings.ReplaceAll(s, op, "\x00")
	}
	parts := strings.Split(s, "\x00")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// firstToken returns the first whitespace-delimited token of a command string.
func firstToken(segment string) string {
	segment = strings.TrimSpace(segment)
	fields := strings.Fields(segment)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

var (
	dollarParenRe = regexp.MustCompile(`\$\(([^)]+)\)`)
	backtickRe    = regexp.MustCompile("`([^`]+)`")
)

// extractSubstitutions returns commands found inside $() and backticks.
func extractSubstitutions(cmd string) []string {
	var subs []string
	for _, m := range dollarParenRe.FindAllStringSubmatch(cmd, -1) {
		subs = append(subs, m[1])
	}
	for _, m := range backtickRe.FindAllStringSubmatch(cmd, -1) {
		subs = append(subs, m[1])
	}
	return subs
}

// InteractiveAllowlist is the set of commands auto-allowed in interactive mode.
// Commands not on this list require user approval (FilterPrompt).
var InteractiveAllowlist = []string{
	"obk", "sqlite3",
	"ls", "cat", "head", "tail", "wc", "sort", "uniq", "diff",
	"find", "grep", "rg",
	"date", "cal", "echo", "printf",
	"git", "tree", "file", "stat", "jq", "which",
}

// DefaultBlocklist is the legacy blocklist used when no Interactor is provided
// (e.g. subagents). Interactive mode now uses InteractiveAllowlist instead.
var DefaultBlocklist = []string{
	// Network
	"curl", "wget", "nc", "ncat", "nmap",
	// Remote access
	"ssh", "scp", "sudo",
	// Permissions
	"chmod", "chown",
	// Shell builtins
	"eval", "exec",
	// Shell wrappers (prevent "bash -c 'curl ...'")
	"bash", "sh", "zsh", "dash", "csh", "ksh", "env", "xargs",
	// Interpreters (prevent "python3 -c 'import urllib...'")
	"python", "python3", "ruby", "perl", "node",
}
