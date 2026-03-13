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
