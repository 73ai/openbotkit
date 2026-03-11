package cli

import (
	"testing"
)

func TestGWSScopesForServices(t *testing.T) {
	scopes := gwsScopesForServices([]string{"calendar", "drive"})
	if len(scopes) != 2 {
		t.Fatalf("got %d scopes, want 2", len(scopes))
	}
	want := map[string]bool{
		"https://www.googleapis.com/auth/calendar": true,
		"https://www.googleapis.com/auth/drive":    true,
	}
	for _, s := range scopes {
		if !want[s] {
			t.Errorf("unexpected scope %q", s)
		}
	}
}

func TestGWSScopesForServices_Unknown(t *testing.T) {
	scopes := gwsScopesForServices([]string{"unknown"})
	if len(scopes) != 0 {
		t.Errorf("got %d scopes for unknown service, want 0", len(scopes))
	}
}

func TestGWSScopesForServices_Empty(t *testing.T) {
	scopes := gwsScopesForServices(nil)
	if len(scopes) != 0 {
		t.Errorf("got %d scopes for empty input, want 0", len(scopes))
	}
}

func TestGWSScopesForServices_AllServices(t *testing.T) {
	all := []string{"calendar", "drive", "docs", "sheets", "tasks", "people"}
	scopes := gwsScopesForServices(all)
	if len(scopes) != 6 {
		t.Fatalf("got %d scopes, want 6", len(scopes))
	}
}
