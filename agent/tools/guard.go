package tools

import (
	"context"
	"fmt"
)

// GuardedWrite requests user approval before executing a write action.
// If approved, it executes the action, notifies "Done.", and returns the result.
// If denied, it notifies "Action not performed." and returns "denied_by_user".
func GuardedWrite(
	ctx context.Context,
	interactor Interactor,
	description string,
	action func() (string, error),
) (string, error) {
	approved, err := interactor.RequestApproval(description)
	if err != nil {
		return "", fmt.Errorf("approval: %w", err)
	}
	if !approved {
		interactor.Notify("Action not performed.")
		return "denied_by_user", nil
	}
	result, err := action()
	if err != nil {
		return "", err
	}
	interactor.Notify("Done.")
	return result, nil
}
