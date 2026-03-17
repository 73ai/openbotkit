package tools

import (
	"github.com/73ai/openbotkit/source/slack"
)

type SlackToolDeps struct {
	Client        slack.API
	Resolver      *slack.Resolver
	Interactor    Interactor
	ApprovalRules *ApprovalRuleSet
}

// SlackResolver returns the shared Resolver, creating one if needed.
func (d SlackToolDeps) SlackResolver() *slack.Resolver {
	if d.Resolver != nil {
		return d.Resolver
	}
	return slack.NewResolver(d.Client)
}

// truncateUTF8 truncates s to at most maxRunes runes, appending "..." if truncated.
func truncateUTF8(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes]) + "..."
}
