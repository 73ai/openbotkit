package tools

import (
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"github.com/priyanshujain/openbotkit/oauth/google"
)

// mockScopeChecker controls which scopes are available.
type mockScopeChecker struct {
	scopes map[string]bool
}

func (m *mockScopeChecker) HasScopes(_ string, required []string) (bool, error) {
	for _, r := range required {
		if !m.scopes[r] {
			return false, nil
		}
	}
	return true, nil
}

func TestGoogleScopeChecker_HasScopes(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "tokens.db")

	store, err := google.NewTokenStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	tok := &oauth2.Token{
		AccessToken:  "tok",
		RefreshToken: "ref",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}
	if err := store.SaveToken("user@example.com", tok, []string{"calendar", "drive"}); err != nil {
		t.Fatal(err)
	}
	store.Close()

	checker := &GoogleScopeChecker{TokenDBPath: dbPath}

	has, err := checker.HasScopes("user@example.com", []string{"calendar"})
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Error("expected to have calendar scope")
	}

	has, err = checker.HasScopes("user@example.com", []string{"gmail"})
	if err != nil {
		t.Fatal(err)
	}
	if has {
		t.Error("should not have gmail scope")
	}
}
