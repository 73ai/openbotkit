package agent

import (
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
	a.compactHistory()
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
	a.compactHistory()

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
	a.compactHistory()

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
	a.compactHistory() // should not panic
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
	a.compactHistory()

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
	a.compactHistory()
	if len(a.history) != 40 {
		t.Errorf("history len = %d, want 40 (at threshold, no compaction)", len(a.history))
	}
}
