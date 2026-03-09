package memory

import "testing"

func TestPreFilter(t *testing.T) {
	messages := []string{
		"ok",
		"thanks",
		"My name is Priyanshu and I work at a startup",
		"yes",
		"I prefer using Go over Python for backend services",
		"hi",
		"sure, got it",
	}

	result := preFilter(messages)

	if len(result) != 2 {
		t.Fatalf("expected 2 messages after filtering, got %d: %v", len(result), result)
	}
	if result[0] != "My name is Priyanshu and I work at a startup" {
		t.Errorf("first = %q", result[0])
	}
	if result[1] != "I prefer using Go over Python for backend services" {
		t.Errorf("second = %q", result[1])
	}
}

func TestPreFilterAllFiltered(t *testing.T) {
	messages := []string{"ok", "yes", "no", "thanks"}
	result := preFilter(messages)
	if len(result) != 0 {
		t.Fatalf("expected 0, got %d", len(result))
	}
}

func TestPreFilterEmpty(t *testing.T) {
	result := preFilter(nil)
	if len(result) != 0 {
		t.Fatalf("expected 0, got %d", len(result))
	}
}

func TestParseExtractionResponse(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{
			name:  "valid JSON array",
			input: `[{"content": "User prefers dark mode", "category": "preference"}]`,
			want:  1,
		},
		{
			name:  "JSON with surrounding text",
			input: "Here are the facts:\n" + `[{"content": "User's name is Priyanshu", "category": "identity"}, {"content": "User likes Go", "category": "preference"}]` + "\nDone.",
			want:  2,
		},
		{
			name:  "empty array",
			input: `[]`,
			want:  0,
		},
		{
			name:  "no JSON",
			input: "No personal facts found.",
			want:  0,
		},
		{
			name:  "empty string",
			input: "",
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			facts, err := parseExtractionResponse(tt.input)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if len(facts) != tt.want {
				t.Fatalf("got %d facts, want %d", len(facts), tt.want)
			}
		})
	}
}

func TestIsAck(t *testing.T) {
	acks := []string{"ok", "thanks", "thank you", "yes", "no", "sure", "got it", "sounds good"}
	for _, a := range acks {
		if !isAck(a) {
			t.Errorf("expected %q to be ack", a)
		}
	}

	nonAcks := []string{"okay let me explain something important here", "my name is priyanshu and i work at a startup"}
	for _, a := range nonAcks {
		if isAck(a) {
			t.Errorf("expected %q to NOT be ack", a)
		}
	}
}
