package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"github.com/priyanshujain/openbotkit/oauth/google"
)

const testCredentials = `{
	"installed": {
		"client_id": "test.apps.googleusercontent.com",
		"client_secret": "secret",
		"auth_uri": "https://accounts.google.com/o/oauth2/auth",
		"token_uri": "https://oauth2.googleapis.com/token",
		"redirect_uris": ["http://localhost"]
	}
}`

func writeTestCreds(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "credentials.json")
	if err := os.WriteFile(path, []byte(testCredentials), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestTokenBridge_Env(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "tokens.db")

	store, err := google.NewTokenStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	tok := &oauth2.Token{
		AccessToken:  "bridge-token-123",
		RefreshToken: "refresh-123",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}
	if err := store.SaveToken("test@example.com", tok, []string{"openid", "email"}); err != nil {
		t.Fatal(err)
	}
	store.Close()

	credPath := writeTestCreds(t, dir)
	g := google.New(google.Config{
		CredentialsFile: credPath,
		TokenDBPath:     dbPath,
	})
	bridge := NewTokenBridge(g, "test@example.com")

	env, err := bridge.Env(context.Background())
	if err != nil {
		t.Fatalf("Env: %v", err)
	}
	if len(env) != 1 {
		t.Fatalf("expected 1 env var, got %d", len(env))
	}
	if !strings.HasPrefix(env[0], "GOOGLE_WORKSPACE_CLI_TOKEN=") {
		t.Errorf("env var = %q", env[0])
	}
	if !strings.Contains(env[0], "bridge-token-123") {
		t.Errorf("env var missing token: %q", env[0])
	}
}
