package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

// GuardOption configures GuardedAction behavior.
type GuardOption func(*guardOpts)

type guardOpts struct {
	rules    *ApprovalRuleSet
	toolName string
	input    json.RawMessage
}

// WithApprovalRules enables session-scoped auto-approve checking.
func WithApprovalRules(rules *ApprovalRuleSet, toolName string, input json.RawMessage) GuardOption {
	return func(o *guardOpts) {
		o.rules = rules
		o.toolName = toolName
		o.input = input
	}
}

// GuardedAction requests user interaction based on risk level before executing.
// RiskLow: notify and auto-approve. RiskMedium/RiskHigh: request approval
// (unless auto-approved by session rules).
func GuardedAction(
	ctx context.Context,
	interactor Interactor,
	risk RiskLevel,
	description string,
	action func() (string, error),
	opts ...GuardOption,
) (string, error) {
	var o guardOpts
	for _, opt := range opts {
		opt(&o)
	}

	if risk == RiskLow {
		if err := interactor.Notify(description); err != nil {
			return "", fmt.Errorf("notify: %w", err)
		}
		return action()
	}

	// Check session auto-approve rules.
	if o.rules != nil && o.rules.Matches(o.toolName, o.input) {
		if err := interactor.Notify("Auto-approved: " + description); err != nil {
			return "", fmt.Errorf("notify auto-approve: %w", err)
		}
		return action()
	}

	approved, err := interactor.RequestApproval(description)
	if err != nil {
		return "", fmt.Errorf("approval: %w", err)
	}
	if !approved {
		if nerr := interactor.Notify("Action not performed."); nerr != nil {
			return "", fmt.Errorf("notify denial: %w", nerr)
		}
		return "denied_by_user", nil
	}

	// Record approval for auto-approve rule generation.
	if o.rules != nil {
		o.rules.RecordApproval(o.toolName, o.input)
		if o.rules.IsRubberStamping() {
			_ = interactor.Notify("You've approved several actions quickly. Take a moment to review.")
		}
	}

	result, err := action()
	if err != nil {
		return "", err
	}
	if nerr := interactor.Notify("Done."); nerr != nil {
		return "", fmt.Errorf("notify completion: %w", nerr)
	}
	return result, nil
}

// GuardedWrite is a backward-compatible wrapper that uses RiskMedium.
func GuardedWrite(
	ctx context.Context,
	interactor Interactor,
	description string,
	action func() (string, error),
) (string, error) {
	return GuardedAction(ctx, interactor, RiskMedium, description, action)
}
