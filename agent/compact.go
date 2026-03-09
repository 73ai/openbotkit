package agent

import (
	"fmt"

	"github.com/priyanshujain/openbotkit/provider"
)

const defaultMaxHistory = 40

func (a *Agent) compactHistory() {
	if len(a.history) <= a.maxHistory {
		return
	}
	keep := a.maxHistory / 2
	if keep < 1 {
		keep = 1
	}
	removed := len(a.history) - keep
	summary := provider.NewTextMessage(provider.RoleUser,
		fmt.Sprintf("[Earlier conversation: %d messages removed]", removed))
	a.history = append([]provider.Message{summary}, a.history[removed:]...)
}
