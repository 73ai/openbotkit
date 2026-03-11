package google

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testCredentials = `{
	"installed": {
		"client_id": "test-client-id.apps.googleusercontent.com",
		"client_secret": "test-secret",
		"auth_uri": "https://accounts.google.com/o/oauth2/auth",
		"token_uri": "https://oauth2.googleapis.com/token",
		"redirect_uris": ["http://localhost"]
	}
}`

func writeTestCredentials(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.json")
	if err := os.WriteFile(path, []byte(testCredentials), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestAuthURL(t *testing.T) {
	credPath := writeTestCredentials(t)
	g := New(Config{CredentialsFile: credPath})

	url, err := g.AuthURL("user@example.com", []string{"https://www.googleapis.com/auth/calendar"}, "test-state-123")
	if err != nil {
		t.Fatalf("AuthURL: %v", err)
	}
	if !strings.Contains(url, "test-state-123") {
		t.Errorf("URL missing state parameter: %s", url)
	}
	if !strings.Contains(url, "include_granted_scopes=true") {
		t.Errorf("URL missing include_granted_scopes: %s", url)
	}
	if !strings.Contains(url, "login_hint=user") {
		t.Errorf("URL missing login_hint: %s", url)
	}
	if !strings.Contains(url, "calendar") {
		t.Errorf("URL missing calendar scope: %s", url)
	}
}

func TestAuthURL_NoAccount(t *testing.T) {
	credPath := writeTestCredentials(t)
	g := New(Config{CredentialsFile: credPath})

	url, err := g.AuthURL("", []string{"https://www.googleapis.com/auth/calendar"}, "state-abc")
	if err != nil {
		t.Fatalf("AuthURL: %v", err)
	}
	if strings.Contains(url, "login_hint") {
		t.Errorf("URL should not contain login_hint for empty account: %s", url)
	}
	if strings.Contains(url, "include_granted_scopes") {
		t.Errorf("URL should not contain include_granted_scopes for empty account: %s", url)
	}
}
