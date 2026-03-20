package usage

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/73ai/openbotkit/provider"
)

func testPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "usage.jsonl")
}

func TestMigrate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "usage.jsonl")
	if err := Migrate(path); err != nil {
		t.Fatalf("migrate: %v", err)
	}
}

func TestMigrateIdempotent(t *testing.T) {
	path := testPath(t)
	if err := Migrate(path); err != nil {
		t.Fatalf("first: %v", err)
	}
	if err := Migrate(path); err != nil {
		t.Fatalf("second: %v", err)
	}
}

func TestRecordAndQuery(t *testing.T) {
	path := testPath(t)

	err := Record(path, UsageRecord{
		Provider:         "anthropic",
		Model:            "claude-sonnet-4-6",
		Channel:          "cli",
		SessionID:        "session-1",
		InputTokens:      1000,
		OutputTokens:     200,
		CacheReadTokens:  800,
		CacheWriteTokens: 100,
	})
	if err != nil {
		t.Fatalf("record: %v", err)
	}

	err = Record(path, UsageRecord{
		Provider:     "openai",
		Model:        "gpt-4o",
		Channel:      "cli",
		SessionID:    "session-1",
		InputTokens:  500,
		OutputTokens: 100,
	})
	if err != nil {
		t.Fatalf("record: %v", err)
	}

	results, err := Query(path, QueryOpts{})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
}

func TestQueryFilterByModel(t *testing.T) {
	path := testPath(t)

	Record(path, UsageRecord{Provider: "anthropic", Model: "claude-sonnet-4-6", InputTokens: 100, OutputTokens: 50})
	Record(path, UsageRecord{Provider: "openai", Model: "gpt-4o", InputTokens: 200, OutputTokens: 80})

	results, err := Query(path, QueryOpts{Model: "gpt-4o"})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Model != "gpt-4o" {
		t.Errorf("model = %q", results[0].Model)
	}
	if results[0].InputTokens != 200 {
		t.Errorf("input tokens = %d", results[0].InputTokens)
	}
}

func TestQueryFilterByDateRange(t *testing.T) {
	path := testPath(t)

	Record(path, UsageRecord{Provider: "anthropic", Model: "claude-sonnet-4-6", InputTokens: 100, OutputTokens: 50})

	future := time.Now().Add(24 * time.Hour)
	results, err := Query(path, QueryOpts{Since: &future})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("got %d results, want 0", len(results))
	}
}

func TestQueryMonthlyGrouping(t *testing.T) {
	path := testPath(t)

	Record(path, UsageRecord{Provider: "anthropic", Model: "claude-sonnet-4-6", InputTokens: 100, OutputTokens: 50})
	Record(path, UsageRecord{Provider: "anthropic", Model: "claude-sonnet-4-6", InputTokens: 200, OutputTokens: 80})

	results, err := Query(path, QueryOpts{GroupBy: "monthly"})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1 (grouped by month)", len(results))
	}
	if results[0].InputTokens != 300 {
		t.Errorf("aggregated input = %d, want 300", results[0].InputTokens)
	}
	if len(results[0].Date) != 7 {
		t.Errorf("expected YYYY-MM date format, got %q", results[0].Date)
	}
}

func TestRecorderIntegration(t *testing.T) {
	path := testPath(t)

	recorder := NewRecorder(path, "anthropic", "cli", "sess-1")
	recorder.RecordUsage("claude-sonnet-4-6", provider.Usage{
		InputTokens:      500,
		OutputTokens:     100,
		CacheReadTokens:  400,
		CacheWriteTokens: 50,
	})

	results, err := Query(path, QueryOpts{})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].InputTokens != 500 {
		t.Errorf("input = %d, want 500", results[0].InputTokens)
	}
	if results[0].CacheReadTokens != 400 {
		t.Errorf("cache_read = %d, want 400", results[0].CacheReadTokens)
	}
}

func TestQueryAggregation(t *testing.T) {
	path := testPath(t)

	for range 3 {
		Record(path, UsageRecord{
			Provider:         "anthropic",
			Model:            "claude-sonnet-4-6",
			InputTokens:      100,
			OutputTokens:     50,
			CacheReadTokens:  80,
			CacheWriteTokens: 10,
		})
	}

	results, err := Query(path, QueryOpts{})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1 (aggregated)", len(results))
	}
	if results[0].InputTokens != 300 {
		t.Errorf("aggregated input = %d, want 300", results[0].InputTokens)
	}
	if results[0].CacheReadTokens != 240 {
		t.Errorf("aggregated cache_read = %d, want 240", results[0].CacheReadTokens)
	}
	if results[0].CallCount != 3 {
		t.Errorf("call count = %d, want 3", results[0].CallCount)
	}
}

func TestQueryEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.jsonl")
	results, err := Query(path, QueryOpts{})
	if err != nil {
		t.Fatalf("query nonexistent: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("got %d results, want 0", len(results))
	}
}
