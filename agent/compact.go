package agent

import (
	"fmt"

	"github.com/priyanshujain/openbotkit/provider"
)

const defaultMaxHistory = 40
const keepMessages = 20

// compactHistory trims history to keepMessages when it exceeds maxHistory,
// prepending a summary placeholder for the removed messages.
func (a *Agent) compactHistory() {
	if len(a.history) <= a.maxHistory {
		return
	}
	keep := keepMessages
	if keep > len(a.history) {
		keep = len(a.history)
	}
	removed := len(a.history) - keep
	if removed <= 0 {
		return
	}
	summary := provider.NewTextMessage(provider.RoleUser,
		fmt.Sprintf("[Earlier conversation: %d messages removed]", removed))
	a.history = append([]provider.Message{summary}, a.history[removed:]...)
}
