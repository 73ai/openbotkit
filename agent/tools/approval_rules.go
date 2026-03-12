package tools

import (
	"encoding/json"
	"sync"
	"time"
)

const (
	autoApproveThreshold = 3
	rubberStampWindow    = 30 * time.Second
	rubberStampThreshold = 5
	maxHistoryLen        = 100
)

// ApprovalRule auto-approves matching tool actions for a session.
type ApprovalRule struct {
	ToolName  string
	Pattern   string
	ExpiresAt time.Time
}

type approvalRecord struct {
	toolName string
	pattern  string
	time     time.Time
}

// ApprovalRuleSet tracks approval history and auto-generates rules
// after repeated approvals of the same action pattern.
type ApprovalRuleSet struct {
	mu      sync.Mutex
	rules   []ApprovalRule
	history []approvalRecord
}

// NewApprovalRuleSet creates a new empty rule set.
func NewApprovalRuleSet() *ApprovalRuleSet {
	return &ApprovalRuleSet{}
}

// Add manually adds an auto-approve rule.
func (s *ApprovalRuleSet) Add(rule ApprovalRule) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rules = append(s.rules, rule)
}

// Matches checks if a tool call matches any active auto-approve rule.
func (s *ApprovalRuleSet) Matches(toolName string, input json.RawMessage) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	pattern := extractPattern(toolName, input)
	now := time.Now()
	for _, r := range s.rules {
		if !r.ExpiresAt.IsZero() && now.After(r.ExpiresAt) {
			continue
		}
		if r.ToolName == toolName && (r.Pattern == "" || r.Pattern == pattern) {
			return true
		}
	}
	return false
}

// RecordApproval records that the user approved an action.
// After autoApproveThreshold approvals of the same (tool, pattern),
// future similar actions auto-approve.
func (s *ApprovalRuleSet) RecordApproval(toolName string, input json.RawMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pattern := extractPattern(toolName, input)
	s.history = append(s.history, approvalRecord{
		toolName: toolName,
		pattern:  pattern,
		time:     time.Now(),
	})
	if len(s.history) > maxHistoryLen {
		s.history = s.history[len(s.history)-maxHistoryLen:]
	}

	count := 0
	for _, h := range s.history {
		if h.toolName == toolName && h.pattern == pattern {
			count++
		}
	}
	if count >= autoApproveThreshold {
		// Check if rule already exists.
		for _, r := range s.rules {
			if r.ToolName == toolName && r.Pattern == pattern {
				return
			}
		}
		s.rules = append(s.rules, ApprovalRule{
			ToolName: toolName,
			Pattern:  pattern,
		})
	}
}

// IsRubberStamping returns true if the user has approved more than
// rubberStampThreshold actions within rubberStampWindow.
func (s *ApprovalRuleSet) IsRubberStamping() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-rubberStampWindow)
	count := 0
	for _, h := range s.history {
		if h.time.After(cutoff) {
			count++
		}
	}
	return count >= rubberStampThreshold
}

// extractPattern derives a tool-specific pattern for grouping similar actions.
func extractPattern(toolName string, input json.RawMessage) string {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(input, &m); err != nil {
		return ""
	}
	switch toolName {
	case "slack_send", "slack_read_channel":
		if ch, ok := m["channel"]; ok {
			var s string
			if json.Unmarshal(ch, &s) == nil {
				return s
			}
		}
	case "gws_execute":
		if cmd, ok := m["command"]; ok {
			var s string
			if json.Unmarshal(cmd, &s) == nil {
				// Use first word as pattern (e.g., "calendar" from "calendar events.list")
				for i, c := range s {
					if c == ' ' {
						return s[:i]
					}
				}
				return s
			}
		}
	}
	return toolName
}
