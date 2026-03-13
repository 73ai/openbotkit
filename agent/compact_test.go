package agent

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/priyanshujain/openbotkit/provider"
)

func TestCompactHistory_BelowThreshold(t *testing.T) {
	a := &Agent{maxHistory: 40}
	for i := range 10 {
		a.history = append(a.history, provider.NewTextMessage(
			provider.RoleUser, fmt.Sprintf("msg %d", i)))
	}
	a.compactHistory(context.Background())
	if len(a.history) != 10 {
		t.Errorf("history len = %d, want 10 (no compaction)", len(a.history))
	}
}

func TestCompactHistory_AboveThreshold(t *testing.T) {
	a := &Agent{maxHistory: 40}
	for i := range 50 {
		a.history = append(a.history, provider.NewTextMessage(
			provider.RoleUser, fmt.Sprintf("msg %d", i)))
	}
	a.compactHistory(context.Background())

	// keep = maxHistory/2 = 20, so result is summary + 20 = 21
	if len(a.history) != 21 {
		t.Fatalf("history len = %d, want 21", len(a.history))
	}

	// First message should be the summary.
	summary := a.history[0].Content[0].Text
	if !strings.Contains(summary, "30 messages removed") {
		t.Errorf("summary = %q, expected '30 messages removed'", summary)
	}

	// Last message should be the original last message.
	last := a.history[20].Content[0].Text
	if last != "msg 49" {
		t.Errorf("last message = %q, want 'msg 49'", last)
	}
}

func TestCompactHistory_SmallMaxHistory(t *testing.T) {
	// maxHistory=6, keep=6/2=3, so 11 messages becomes summary + 3 = 4.
	a := &Agent{maxHistory: 6}
	for i := range 11 {
		a.history = append(a.history, provider.NewTextMessage(
			provider.RoleUser, fmt.Sprintf("msg %d", i)))
	}
	a.compactHistory(context.Background())

	if len(a.history) != 4 {
		t.Errorf("history len = %d, want 4 (summary + 3 kept)", len(a.history))
	}

	// Last message should be the original last message.
	last := a.history[3].Content[0].Text
	if last != "msg 10" {
		t.Errorf("last message = %q, want 'msg 10'", last)
	}
}

func TestCompactHistory_EmptyHistory(t *testing.T) {
	a := &Agent{maxHistory: 40}
	a.compactHistory(context.Background()) // should not panic
	if len(a.history) != 0 {
		t.Errorf("history len = %d, want 0", len(a.history))
	}
}

func TestCompactHistory_MaxHistoryOne(t *testing.T) {
	// maxHistory=1, keep=1/2=0, guard sets keep=1.
	// 5 messages → summary + 1 kept = 2.
	a := &Agent{maxHistory: 1}
	for i := range 5 {
		a.history = append(a.history, provider.NewTextMessage(
			provider.RoleUser, fmt.Sprintf("msg %d", i)))
	}
	a.compactHistory(context.Background())

	if len(a.history) != 2 {
		t.Fatalf("history len = %d, want 2 (summary + 1 kept)", len(a.history))
	}

	last := a.history[1].Content[0].Text
	if last != "msg 4" {
		t.Errorf("last message = %q, want 'msg 4'", last)
	}
}

func TestCompactHistory_ExactThreshold(t *testing.T) {
	a := &Agent{maxHistory: 40}
	for i := range 40 {
		a.history = append(a.history, provider.NewTextMessage(
			provider.RoleUser, fmt.Sprintf("msg %d", i)))
	}
	a.compactHistory(context.Background())
	if len(a.history) != 40 {
		t.Errorf("history len = %d, want 40 (at threshold, no compaction)", len(a.history))
	}
}

// mockSummarizer is a test double for the Summarizer interface.
type mockSummarizer struct {
	result string
	err    error
	called bool
}

func (m *mockSummarizer) Summarize(_ context.Context, _ []provider.Message) (string, error) {
	m.called = true
	return m.result, m.err
}

func TestCompactHistory_TokenTrigger_NoSummarizer(t *testing.T) {
	a := &Agent{
		maxHistory:          10,
		contextWindow:       200000,
		compactionThreshold: 0.30,
		lastInputTokens:     70000, // > 200000 * 0.30 = 60000
	}
	for i := range 20 {
		a.history = append(a.history, provider.NewTextMessage(
			provider.RoleUser, fmt.Sprintf("msg %d", i)))
	}
	a.compactHistory(context.Background())

	// Falls through to truncateHistory since no summarizer
	// keep=10/2=5, so 20→summary+5=6
	if len(a.history) != 6 {
		t.Errorf("expected 6 after truncation, history len = %d", len(a.history))
	}
	summary := a.history[0].Content[0].Text
	if !strings.Contains(summary, "messages removed") {
		t.Errorf("expected truncation marker, got %q", summary)
	}
}

func TestCompactHistory_TokenTrigger_WithSummarizer(t *testing.T) {
	ms := &mockSummarizer{result: "summary text"}
	a := &Agent{
		maxHistory:          40,
		contextWindow:       200000,
		compactionThreshold: 0.30,
		lastInputTokens:     70000,
		summarizer:          ms,
	}
	for i := range 20 {
		a.history = append(a.history, provider.NewTextMessage(
			provider.RoleUser, fmt.Sprintf("msg %d", i)))
	}
	a.compactHistory(context.Background())

	if !ms.called {
		t.Fatal("summarizer was not called")
	}
	// Split: 10 summarized + 10 kept → 1 summary + 10 = 11
	if len(a.history) != 11 {
		t.Fatalf("history len = %d, want 11", len(a.history))
	}
	if !strings.Contains(a.history[0].Content[0].Text, "summary text") {
		t.Errorf("first message = %q, expected summary", a.history[0].Content[0].Text)
	}
}

func TestCompactHistory_BelowTokenThreshold_NoCompaction(t *testing.T) {
	a := &Agent{
		maxHistory:          40,
		contextWindow:       200000,
		compactionThreshold: 0.30,
		lastInputTokens:     10000, // < 60000
	}
	for i := range 10 {
		a.history = append(a.history, provider.NewTextMessage(
			provider.RoleUser, fmt.Sprintf("msg %d", i)))
	}
	a.compactHistory(context.Background())

	if len(a.history) != 10 {
		t.Errorf("history len = %d, want 10 (no compaction)", len(a.history))
	}
}

func TestCompactHistory_SummarizerError_FallbackToTruncation(t *testing.T) {
	ms := &mockSummarizer{err: fmt.Errorf("LLM error")}
	a := &Agent{
		maxHistory:          40,
		contextWindow:       200000,
		compactionThreshold: 0.30,
		lastInputTokens:     70000,
		summarizer:          ms,
	}
	for i := range 50 {
		a.history = append(a.history, provider.NewTextMessage(
			provider.RoleUser, fmt.Sprintf("msg %d", i)))
	}
	a.compactHistory(context.Background())

	if !ms.called {
		t.Fatal("summarizer should have been called")
	}
	if len(a.history) >= 50 {
		t.Errorf("expected truncation, history len = %d", len(a.history))
	}
	summary := a.history[0].Content[0].Text
	if !strings.Contains(summary, "messages removed") {
		t.Errorf("expected truncation marker, got %q", summary)
	}
}

func TestCompactHistory_NoContextWindow_MessageCountFallback(t *testing.T) {
	a := &Agent{maxHistory: 10}
	for i := range 20 {
		a.history = append(a.history, provider.NewTextMessage(
			provider.RoleUser, fmt.Sprintf("msg %d", i)))
	}
	a.compactHistory(context.Background())

	// keep=10/2=5, result = summary + 5 = 6
	if len(a.history) != 6 {
		t.Errorf("history len = %d, want 6", len(a.history))
	}
}

func TestCompactHistory_TokenTrigger_FewMessages(t *testing.T) {
	ms := &mockSummarizer{result: "brief summary"}
	a := &Agent{
		maxHistory:          40,
		contextWindow:       200000,
		compactionThreshold: 0.30,
		lastInputTokens:     70000,
		summarizer:          ms,
	}
	a.history = []provider.Message{
		provider.NewTextMessage(provider.RoleUser, "msg1"),
		provider.NewTextMessage(provider.RoleAssistant, "msg2"),
	}
	a.compactHistory(context.Background())

	if !ms.called {
		t.Fatal("summarizer should be called")
	}
	// Split: 1 summarized + 1 kept → 1 summary + 1 = 2
	if len(a.history) != 2 {
		t.Fatalf("history len = %d, want 2", len(a.history))
	}
	if !strings.Contains(a.history[0].Content[0].Text, "brief summary") {
		t.Errorf("first message = %q, expected summary", a.history[0].Content[0].Text)
	}
}

func TestCompactHistory_EmptySummary_FallbackToTruncation(t *testing.T) {
	ms := &mockSummarizer{result: ""}
	a := &Agent{
		maxHistory:          40,
		contextWindow:       200000,
		compactionThreshold: 0.30,
		lastInputTokens:     70000,
		summarizer:          ms,
	}
	for i := range 50 {
		a.history = append(a.history, provider.NewTextMessage(
			provider.RoleUser, fmt.Sprintf("msg %d", i)))
	}
	a.compactHistory(context.Background())

	if !ms.called {
		t.Fatal("summarizer should have been called")
	}
	// Empty summary → falls back to truncation
	if len(a.history) >= 50 {
		t.Errorf("expected truncation, history len = %d", len(a.history))
	}
	summary := a.history[0].Content[0].Text
	if !strings.Contains(summary, "messages removed") {
		t.Errorf("expected truncation marker, got %q", summary)
	}
}
