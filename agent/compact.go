package agent

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/priyanshujain/openbotkit/provider"
)

const defaultMaxHistory = 40

func (a *Agent) compactHistory(ctx context.Context) {
	if a.tokenThresholdExceeded() {
		if a.summarizer != nil {
			if a.compactWithSummary(ctx) {
				return
			}
		}
		a.truncateHistory()
		return
	}

	if len(a.history) > a.maxHistory {
		a.truncateHistory()
	}
}

func (a *Agent) tokenThresholdExceeded() bool {
	if a.contextWindow <= 0 || a.compactionThreshold <= 0 || a.lastInputTokens <= 0 {
		return false
	}
	return a.lastInputTokens > int(float64(a.contextWindow)*a.compactionThreshold)
}

func (a *Agent) compactWithSummary(ctx context.Context) bool {
	if len(a.history) < 2 {
		return false
	}
	splitPoint := len(a.history) / 2
	olderHalf := a.history[:splitPoint]
	recentHalf := a.history[splitPoint:]

	text, err := a.summarizer.Summarize(ctx, olderHalf)
	if err != nil {
		slog.Warn("compaction summarizer failed, falling back to truncation", "error", err)
		return false
	}
	if text == "" {
		slog.Warn("compaction summarizer returned empty summary, falling back to truncation")
		return false
	}

	summary := provider.NewTextMessage(provider.RoleUser,
		fmt.Sprintf("[Conversation summary: %s]", text))
	a.history = append([]provider.Message{summary}, recentHalf...)
	return true
}

func (a *Agent) truncateHistory() {
	keep := a.maxHistory / 2
	if keep < 1 {
		keep = 1
	}
	removed := len(a.history) - keep
	summary := provider.NewTextMessage(provider.RoleUser,
		fmt.Sprintf("[Earlier conversation: %d messages removed]", removed))
	a.history = append([]provider.Message{summary}, a.history[removed:]...)
}
