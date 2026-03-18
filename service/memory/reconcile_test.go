package memory

import (
	"context"
	"fmt"
	"testing"
)

func TestReconcileNoExisting(t *testing.T) {
	db := testDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	candidates := []CandidateFact{
		{Content: "User prefers dark mode", Category: "preference"},
		{Content: "User's name is Priyanshu", Category: "identity"},
	}

	// With no existing memories, LLM is not needed (all ADD).
	result, err := Reconcile(context.Background(), db, nil, candidates)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	if result.Added != 2 {
		t.Errorf("added = %d, want 2", result.Added)
	}

	count, _ := Count(db)
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestReconcileWithExistingNOOP(t *testing.T) {
	db := testDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Seed an existing memory.
	Add(db, "User prefers dark mode", CategoryPreference, "manual", "")

	llm := &mockLLM{
		response: `{"action": "NOOP"}`,
	}

	candidates := []CandidateFact{
		{Content: "User prefers dark mode", Category: "preference"},
	}

	result, err := Reconcile(context.Background(), db, llm, candidates)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	if result.Skipped != 1 {
		t.Errorf("skipped = %d, want 1", result.Skipped)
	}
	if result.Added != 0 {
		t.Errorf("added = %d, want 0", result.Added)
	}

	// DB should still have just 1 memory.
	count, _ := Count(db)
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}

func TestReconcileWithExistingUPDATE(t *testing.T) {
	db := testDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	id, _ := Add(db, "User prefers light mode", CategoryPreference, "manual", "")

	llm := &mockLLM{
		response: fmt.Sprintf(`{"action": "UPDATE", "id": %d, "content": "User prefers dark mode"}`, id),
	}

	candidates := []CandidateFact{
		{Content: "User now prefers dark mode", Category: "preference"},
	}

	result, err := Reconcile(context.Background(), db, llm, candidates)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	if result.Updated != 1 {
		t.Errorf("updated = %d, want 1", result.Updated)
	}

	// Verify the memory was updated.
	m, _ := Get(db, id)
	if m.Content != "User prefers dark mode" {
		t.Errorf("content = %q, want 'User prefers dark mode'", m.Content)
	}
}

func TestReconcileWithExistingDELETE(t *testing.T) {
	db := testDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	id, _ := Add(db, "User works at Acme Corp", CategoryIdentity, "manual", "")

	llm := &mockLLM{
		response: fmt.Sprintf(`{"action": "DELETE", "id": %d}`, id),
	}

	candidates := []CandidateFact{
		{Content: "User left Acme Corp", Category: "identity"},
	}

	result, err := Reconcile(context.Background(), db, llm, candidates)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	if result.Deleted != 1 {
		t.Errorf("deleted = %d, want 1", result.Deleted)
	}

	count, _ := Count(db)
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}
}

func TestReconcileWithExistingADD(t *testing.T) {
	db := testDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Seed existing memory that shares keywords.
	Add(db, "User prefers dark mode", CategoryPreference, "manual", "")

	llm := &mockLLM{
		response: `{"action": "ADD"}`,
	}

	candidates := []CandidateFact{
		{Content: "User prefers vim keybindings in dark mode editors", Category: "preference"},
	}

	result, err := Reconcile(context.Background(), db, llm, candidates)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	if result.Added != 1 {
		t.Errorf("added = %d, want 1", result.Added)
	}

	count, _ := Count(db)
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestReconcileLLMError(t *testing.T) {
	db := testDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	Add(db, "User prefers dark mode", CategoryPreference, "manual", "")

	llm := &mockLLM{err: fmt.Errorf("API error")}

	candidates := []CandidateFact{
		{Content: "User prefers dark mode in editors", Category: "preference"},
	}

	result, err := Reconcile(context.Background(), db, llm, candidates)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	if result.Skipped != 1 {
		t.Errorf("skipped = %d, want 1 (LLM error should skip)", result.Skipped)
	}
}

func TestParseReconcileResponse(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		action string
	}{
		{
			name:   "ADD",
			input:  `{"action": "ADD", "id": 0, "content": ""}`,
			action: "ADD",
		},
		{
			name:   "UPDATE with surrounding text",
			input:  `Based on the analysis: {"action": "UPDATE", "id": 5, "content": "User now prefers dark mode"} end`,
			action: "UPDATE",
		},
		{
			name:   "NOOP",
			input:  `{"action": "NOOP"}`,
			action: "NOOP",
		},
		{
			name:   "no JSON",
			input:  "I don't know",
			action: "NOOP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := parseReconcileResponse(tt.input)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if d.Action != tt.action {
				t.Errorf("action = %q, want %q", d.Action, tt.action)
			}
		})
	}
}

func TestExtractKeywords(t *testing.T) {
	kw := extractKeywords("User prefers Go over Python for backend services")

	if len(kw) == 0 {
		t.Fatal("expected keywords")
	}
	if len(kw) > 3 {
		t.Fatalf("expected at most 3 keywords, got %d", len(kw))
	}

	for _, w := range kw {
		if w == "user" {
			t.Error("should skip stop word 'user'")
		}
	}
}

func TestDedup(t *testing.T) {
	existing := []Memory{{ID: 1, Content: "fact one"}}
	newItems := []Memory{{ID: 1, Content: "fact one"}, {ID: 2, Content: "fact two"}}

	result := dedup(existing, newItems)
	if len(result) != 2 {
		t.Fatalf("expected 2, got %d", len(result))
	}
}
