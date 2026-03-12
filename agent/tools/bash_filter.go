package tools

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// CommandFilter validates shell commands against an allowlist or blocklist.
type CommandFilter struct {
	allowed []string // if set, only these prefixes pass
	blocked []string // if set, these prefixes are rejected
}

// NewAllowlistFilter creates a filter that only permits commands
// whose first token matches one of the given prefixes.
func NewAllowlistFilter(prefixes []string) *CommandFilter {
	return &CommandFilter{allowed: prefixes}
}

// NewBlocklistFilter creates a filter that rejects commands
// whose first token matches any of the given prefixes.
func NewBlocklistFilter(prefixes []string) *CommandFilter {
	return &CommandFilter{blocked: prefixes}
}

// Check validates the given command string. It splits on shell
// operators (|, &&, ;, ||) and checks each segment. It also
// detects command substitution via $() and backticks.
func (f *CommandFilter) Check(command string) error {
	if f == nil {
		return nil
	}

	segments := splitShellSegments(command)
	for _, seg := range segments {
		if err := f.checkSegment(seg); err != nil {
			return err
		}
	}

	// Check inside $() and backtick substitutions.
	for _, sub := range extractSubstitutions(command) {
		if err := f.Check(sub); err != nil {
			return fmt.Errorf("in command substitution: %w", err)
		}
	}

	return nil
}

// basename strips directory components so "/usr/bin/curl" → "curl".
func basename(token string) string {
	return filepath.Base(token)
}

// checkSegment validates a single command segment.
// Allowlist: only the first token must match.
// Blocklist: every token is checked to catch wrappers like "env curl".
func (f *CommandFilter) checkSegment(seg string) error {
	fields := strings.Fields(strings.TrimSpace(seg))
	if len(fields) == 0 {
		return nil
	}
	if len(f.allowed) > 0 {
		return f.checkToken(fields[0])
	}
	for _, tok := range fields {
		if err := f.checkToken(tok); err != nil {
			return err
		}
	}
	return nil
}

func (f *CommandFilter) checkToken(token string) error {
	base := basename(token)
	if len(f.allowed) > 0 {
		for _, prefix := range f.allowed {
			if base == prefix {
				return nil
			}
		}
		return fmt.Errorf("command %q not in allowlist", token)
	}
	for _, prefix := range f.blocked {
		if base == prefix {
			return fmt.Errorf("command %q is blocked", token)
		}
	}
	return nil
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

// DefaultBlocklist is the default set of blocked commands for interactive mode.
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
