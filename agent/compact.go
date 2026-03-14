package agent

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/priyanshujain/openbotkit/provider"
)

const defaultMaxHistory = 40

const microcompactAge = 3

func (a *Agent) microcompact() {
	if len(a.history) < microcompactAge*2 {
		return
	}
	cutoff := len(a.history) - microcompactAge*2
	for i := 0; i < cutoff; i++ {
		msg := &a.history[i]
		if msg.Role != provider.RoleUser {
			continue
		}
		for j := range msg.Content {
			block := &msg.Content[j]
			if block.Type != provider.ContentToolResult || block.ToolResult == nil {
				continue
			}
			if len(block.ToolResult.Content) <= 200 {
				continue
			}
			if idx := strings.LastIndex(block.ToolResult.Content, "[Full output: "); idx >= 0 {
				ref := block.ToolResult.Content[idx:]
				block.ToolResult.Content = fmt.Sprintf("[Previous: used %s] %s", block.ToolResult.Name, ref)
			} else {
				block.ToolResult.Content = fmt.Sprintf("[Previous: used %s]", block.ToolResult.Name)
			}
		}
	}
}

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
