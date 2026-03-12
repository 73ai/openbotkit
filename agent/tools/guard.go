package tools

import (
	"context"
	"fmt"
)

// GuardedAction requests user interaction based on risk level before executing.
// RiskLow: notify and auto-approve. RiskMedium/RiskHigh: request approval.
func GuardedAction(
	ctx context.Context,
	interactor Interactor,
	risk RiskLevel,
	description string,
	action func() (string, error),
) (string, error) {
	if risk == RiskLow {
		if err := interactor.Notify(description); err != nil {
			return "", fmt.Errorf("notify: %w", err)
		}
		result, err := action()
		if err != nil {
			return "", err
		}
		return result, nil
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
