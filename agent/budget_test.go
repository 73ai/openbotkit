package agent

import (
	"testing"

	"github.com/73ai/openbotkit/provider"
)

type mockRecorder struct {
	calls int
}

func (m *mockRecorder) RecordUsage(_ string, _ provider.Usage) {
	m.calls++
}

func TestBudgetTracker_UnderBudget(t *testing.T) {
	bt := NewBudgetTracker(1.0, nil)
	bt.RecordUsage("claude-sonnet-4-6", provider.Usage{InputTokens: 1000, OutputTokens: 1000})
	if err := bt.CheckBudget(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBudgetTracker_ExceedsBudget(t *testing.T) {
	bt := NewBudgetTracker(0.01, nil)
	// 1M tokens of sonnet = $18, well over $0.01
	bt.RecordUsage("claude-sonnet-4-6", provider.Usage{InputTokens: 1_000_000, OutputTokens: 1_000_000})
	if err := bt.CheckBudget(); err == nil {
		t.Error("expected budget exceeded error")
	}
}

func TestBudgetTracker_Unlimited(t *testing.T) {
	bt := NewBudgetTracker(0, nil)
	bt.RecordUsage("claude-opus-4-6", provider.Usage{InputTokens: 10_000_000, OutputTokens: 10_000_000})
	if err := bt.CheckBudget(); err != nil {
		t.Errorf("unlimited budget should never error: %v", err)
	}
}

func TestBudgetTracker_ChainsInnerRecorder(t *testing.T) {
	inner := &mockRecorder{}
	bt := NewBudgetTracker(1.0, inner)
	bt.RecordUsage("claude-sonnet-4-6", provider.Usage{InputTokens: 100})
	bt.RecordUsage("claude-sonnet-4-6", provider.Usage{InputTokens: 200})
	if inner.calls != 2 {
		t.Errorf("inner.calls = %d, want 2", inner.calls)
	}
}

func TestBudgetTracker_Total(t *testing.T) {
	bt := NewBudgetTracker(1.0, nil)
	if bt.Total() != 0 {
		t.Errorf("initial total = %f, want 0", bt.Total())
	}
	bt.RecordUsage("claude-sonnet-4-6", provider.Usage{InputTokens: 1000, OutputTokens: 1000})
	if bt.Total() <= 0 {
		t.Error("expected positive total after recording usage")
	}
}
