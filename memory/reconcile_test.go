package memory

import "testing"

func TestReconcileNoExisting(t *testing.T) {
	db := testDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// With no existing memories and no router (candidates should all be ADD).
	candidates := []CandidateFact{
		{Content: "User prefers dark mode", Category: "preference"},
		{Content: "User's name is Priyanshu", Category: "identity"},
	}

	result, err := Reconcile(nil, db, nil, candidates)
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

	// Should skip "user" (stop word) and short words.
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
