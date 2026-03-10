package spectest

import (
	"strings"
	"testing"
)

func AssertNotEmpty(t *testing.T, response string) {
	t.Helper()
	if strings.TrimSpace(response) == "" {
		t.Fatal("expected non-empty response")
	}
}

func AssertContains(t *testing.T, response string, substrings ...string) {
	t.Helper()
	lower := strings.ToLower(response)
	for _, s := range substrings {
		if !strings.Contains(lower, strings.ToLower(s)) {
			t.Errorf("expected response to contain %q, got:\n%s", s, response)
		}
	}
}
