package agent

import (
	"fmt"
	"sync"

	"github.com/73ai/openbotkit/provider"
)

// BudgetChecker checks whether the accumulated cost has exceeded a budget.
type BudgetChecker interface {
	CheckBudget() error
}

// BudgetTracker wraps a UsageRecorder and accumulates cost per call.
// It implements both UsageRecorder and BudgetChecker.
type BudgetTracker struct {
	maxBudget float64
	inner     UsageRecorder
	mu        sync.Mutex
	total     float64
}

// NewBudgetTracker creates a tracker that enforces a cost budget.
// If maxBudget is 0, budget checking is disabled (unlimited).
func NewBudgetTracker(maxBudget float64, inner UsageRecorder) *BudgetTracker {
	return &BudgetTracker{maxBudget: maxBudget, inner: inner}
}

func (bt *BudgetTracker) RecordUsage(model string, usage provider.Usage) {
	cost := provider.EstimateCost(model, usage)
	bt.mu.Lock()
	bt.total += cost
	bt.mu.Unlock()
	if bt.inner != nil {
		bt.inner.RecordUsage(model, usage)
	}
}

func (bt *BudgetTracker) CheckBudget() error {
	if bt.maxBudget <= 0 {
		return nil
	}
	bt.mu.Lock()
	total := bt.total
	bt.mu.Unlock()
	if total >= bt.maxBudget {
		return fmt.Errorf("budget exceeded: $%.4f spent of $%.4f limit", total, bt.maxBudget)
	}
	return nil
}

// Total returns the accumulated cost so far.
func (bt *BudgetTracker) Total() float64 {
	bt.mu.Lock()
	defer bt.mu.Unlock()
	return bt.total
}
