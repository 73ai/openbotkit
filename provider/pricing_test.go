package provider

import "testing"

func TestEstimateCost_KnownModel(t *testing.T) {
	usage := Usage{InputTokens: 1_000_000, OutputTokens: 1_000_000}
	cost := EstimateCost("claude-sonnet-4-6", usage)
	// 3.0 input + 15.0 output = 18.0
	if cost != 18.0 {
		t.Errorf("cost = %f, want 18.0", cost)
	}
}

func TestEstimateCost_UnknownModel(t *testing.T) {
	usage := Usage{InputTokens: 1000, OutputTokens: 1000}
	cost := EstimateCost("unknown-model-v1", usage)
	if cost != 0 {
		t.Errorf("cost = %f, want 0", cost)
	}
}

func TestEstimateCost_PrefixMatching(t *testing.T) {
	usage := Usage{InputTokens: 1_000_000, OutputTokens: 1_000_000}
	cost := EstimateCost("claude-sonnet-4-6-20260101", usage)
	if cost != 18.0 {
		t.Errorf("prefix match cost = %f, want 18.0", cost)
	}
}

func TestEstimateCost_WithCache(t *testing.T) {
	usage := Usage{
		InputTokens:     1_000_000,
		OutputTokens:    500_000,
		CacheReadTokens: 200_000,
	}
	cost := EstimateCost("claude-sonnet-4-6", usage)
	// non-cached input: 800k * 3.0/M = 2.4
	// output: 500k * 15.0/M = 7.5
	// cache read: 200k * 0.30/M = 0.06
	// total = 9.96
	if cost != 9.96 {
		t.Errorf("cost = %f, want 9.96", cost)
	}
}
