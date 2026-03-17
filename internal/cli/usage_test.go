package cli

import (
	"testing"

	usagesrc "github.com/73ai/openbotkit/source/usage"
)

func TestFormatTokens(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0"},
		{500, "500"},
		{1000, "1.0K"},
		{1500, "1.5K"},
		{45230, "45.2K"},
		{1000000, "1.0M"},
		{1500000, "1.5M"},
	}
	for _, tt := range tests {
		got := formatTokens(tt.input)
		if got != tt.want {
			t.Errorf("formatTokens(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestEstimateCost_KnownModel(t *testing.T) {
	r := usagesrc.AggregatedUsage{
		Model:            "claude-sonnet-4-6",
		InputTokens:      100_000,
		OutputTokens:     20_000,
		CacheReadTokens:  80_000,
		CacheWriteTokens: 5_000,
	}
	cost := estimateCost(r)
	if cost <= 0 {
		t.Errorf("expected positive cost, got %f", cost)
	}
	// With 80K cache reads at 0.10x rate, cost should be lower than full rate.
	fullRate := usagesrc.AggregatedUsage{
		Model:        "claude-sonnet-4-6",
		InputTokens:  100_000,
		OutputTokens: 20_000,
	}
	fullCost := estimateCost(fullRate)
	if cost >= fullCost {
		t.Errorf("cached cost (%f) should be less than full cost (%f)", cost, fullCost)
	}
}

func TestEstimateCost_UnknownModel(t *testing.T) {
	r := usagesrc.AggregatedUsage{
		Model:       "unknown-model-v42",
		InputTokens: 100_000,
	}
	cost := estimateCost(r)
	if cost != 0 {
		t.Errorf("unknown model should return 0 cost, got %f", cost)
	}
}

func TestEstimateCost_PrefixMatchLongestWins(t *testing.T) {
	// "gpt-4o-mini-2025" should match "gpt-4o-mini" pricing, not "gpt-4o".
	r := usagesrc.AggregatedUsage{
		Model:        "gpt-4o-mini-2025-01-01",
		InputTokens:  1_000_000,
		OutputTokens: 0,
	}
	cost := estimateCost(r)
	// gpt-4o-mini input rate = $0.15/M → cost should be $0.15
	if cost != 0.15 {
		t.Errorf("expected $0.15 (gpt-4o-mini rate), got $%.2f", cost)
	}
}

func TestEstimateCost_OpenAI(t *testing.T) {
	r := usagesrc.AggregatedUsage{
		Model:           "gpt-4o",
		InputTokens:     50_000,
		OutputTokens:    10_000,
		CacheReadTokens: 40_000,
	}
	cost := estimateCost(r)
	if cost <= 0 {
		t.Errorf("expected positive cost for gpt-4o, got %f", cost)
	}
}
