package provider

import "testing"

func TestDefaultContextWindow_Claude(t *testing.T) {
	if got := DefaultContextWindow("claude-opus-4-6"); got != 200000 {
		t.Errorf("claude-opus-4-6 = %d, want 200000", got)
	}
}

func TestDefaultContextWindow_GPT(t *testing.T) {
	if got := DefaultContextWindow("gpt-4o"); got != 128000 {
		t.Errorf("gpt-4o = %d, want 128000", got)
	}
}

func TestDefaultContextWindow_Gemini(t *testing.T) {
	if got := DefaultContextWindow("gemini-2.5-pro"); got != 1048576 {
		t.Errorf("gemini-2.5-pro = %d, want 1048576", got)
	}
}

func TestDefaultContextWindow_Unknown(t *testing.T) {
	if got := DefaultContextWindow("unknown-model"); got != 0 {
		t.Errorf("unknown = %d, want 0", got)
	}
}

func TestDefaultContextWindow_PrefixMatch(t *testing.T) {
	if got := DefaultContextWindow("claude-opus-4-6-20260301"); got != 200000 {
		t.Errorf("claude-opus-4-6-20260301 = %d, want 200000", got)
	}
	if got := DefaultContextWindow("gemini-2.5-pro-preview"); got != 1048576 {
		t.Errorf("gemini-2.5-pro-preview = %d, want 1048576", got)
	}
}

func TestDefaultContextWindow_NestedModelID(t *testing.T) {
	// OpenRouter-style nested model IDs like "anthropic/claude-haiku-4-5".
	if got := DefaultContextWindow("anthropic/claude-haiku-4-5"); got != 200000 {
		t.Errorf("anthropic/claude-haiku-4-5 = %d, want 200000", got)
	}
	if got := DefaultContextWindow("anthropic/claude-sonnet-4-6"); got != 200000 {
		t.Errorf("anthropic/claude-sonnet-4-6 = %d, want 200000", got)
	}
	if got := DefaultContextWindow("google/gemini-2.0-flash-lite"); got != 1048576 {
		t.Errorf("google/gemini-2.0-flash-lite = %d, want 1048576", got)
	}
	if got := DefaultContextWindow("mistralai/mistral-medium-3.1"); got != 131072 {
		t.Errorf("mistralai/mistral-medium-3.1 = %d, want 131072", got)
	}
}

func TestDefaultContextWindow_GroqModels(t *testing.T) {
	models := map[string]int{
		"llama-3.1-8b-instant":      131072,
		"llama-3.3-70b-versatile":   131072,
		"llama-4-scout-17b-16e":     131072,
		"llama-4-maverick-17b-128e": 131072,
	}
	for model, want := range models {
		if got := DefaultContextWindow(model); got != want {
			t.Errorf("%s = %d, want %d", model, got, want)
		}
	}
}
