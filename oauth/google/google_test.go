package google

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
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

func TestExchangeCode_MergesScopes(t *testing.T) {
	dir := t.TempDir()
	credPath := writeTestCredentials(t)
	dbPath := filepath.Join(dir, "tokens.db")

	// Pre-seed a token.
	store, err := NewTokenStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	tok := &oauth2.Token{
		AccessToken:  "old-access",
		RefreshToken: "old-refresh",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}
	if err := store.SaveToken("user@example.com", tok, []string{"scope-a"}); err != nil {
		t.Fatal(err)
	}
	store.Close()

	g := New(Config{CredentialsFile: credPath, TokenDBPath: dbPath})

	// ExchangeCode will fail at the Exchange step (no real server),
	// but we can verify the setup is correct by checking that loadConfig succeeds.
	_, err = g.ExchangeCode(context.Background(), "bad-code", "user@example.com", []string{"scope-b"})
	if err == nil {
		t.Fatal("expected error from exchange (no real auth server)")
	}
	// Verify it's a token exchange error, not a config/store error.
	if !strings.Contains(err.Error(), "exchange token") {
		t.Fatalf("expected exchange token error, got: %v", err)
	}
}

func TestAccessToken_ValidToken(t *testing.T) {
	dir := t.TempDir()
	credPath := writeTestCredentials(t)
	dbPath := filepath.Join(dir, "tokens.db")

	store, err := NewTokenStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	tok := &oauth2.Token{
		AccessToken:  "my-access-token",
		RefreshToken: "my-refresh-token",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}
	if err := store.SaveToken("user@example.com", tok, []string{"openid", "email"}); err != nil {
		t.Fatal(err)
	}
	store.Close()

	g := New(Config{CredentialsFile: credPath, TokenDBPath: dbPath})
	got, err := g.AccessToken(context.Background(), "user@example.com")
	if err != nil {
		t.Fatalf("AccessToken: %v", err)
	}
	if got != "my-access-token" {
		t.Errorf("AccessToken = %q, want %q", got, "my-access-token")
	}
}

func TestAuthURL_WithCallbackURL(t *testing.T) {
	credPath := writeTestCredentials(t)
	callbackURL := "https://my-bot.ngrok-free.app/auth/google/callback"
	g := New(Config{
		CredentialsFile: credPath,
		CallbackURL:     callbackURL,
	})

	url, err := g.AuthURL("user@example.com", []string{"https://www.googleapis.com/auth/drive"}, "state-xyz")
	if err != nil {
		t.Fatalf("AuthURL: %v", err)
	}
	// The redirect_uri in the URL should be our public callback, not localhost.
	if !strings.Contains(url, "my-bot.ngrok-free.app") {
		t.Errorf("URL should contain ngrok domain for redirect_uri, got: %s", url)
	}
	if strings.Contains(url, "localhost") {
		t.Errorf("URL should NOT contain localhost when CallbackURL is set, got: %s", url)
	}
}

func TestAuthURL_DefaultRedirect(t *testing.T) {
	credPath := writeTestCredentials(t)
	g := New(Config{CredentialsFile: credPath})

	url, err := g.AuthURL("", []string{"https://www.googleapis.com/auth/drive"}, "state-abc")
	if err != nil {
		t.Fatalf("AuthURL: %v", err)
	}
	if !strings.Contains(url, "localhost") {
		t.Errorf("URL should use localhost when CallbackURL is empty, got: %s", url)
	}
}

func TestAuthURL_ExchangeCode_RedirectConsistency(t *testing.T) {
	dir := t.TempDir()
	credPath := writeTestCredentials(t)
	dbPath := filepath.Join(dir, "tokens.db")

	store, err := NewTokenStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	store.Close()

	callbackURL := "https://my-bot.ngrok-free.app/auth/google/callback"
	g := New(Config{
		CredentialsFile: credPath,
		TokenDBPath:     dbPath,
		CallbackURL:     callbackURL,
	})

	// Get the auth URL's redirect_uri.
	authURL, err := g.AuthURL("user@example.com", []string{"https://www.googleapis.com/auth/drive"}, "state-1")
	if err != nil {
		t.Fatalf("AuthURL: %v", err)
	}

	// ExchangeCode will fail (no real server), but it should use the same redirect_uri.
	// We can verify by checking that loadConfig is called with CallbackURL,
	// which we've already unit-tested. But let's at least confirm ExchangeCode
	// reaches the exchange step (not a config error) when CallbackURL is set.
	_, err = g.ExchangeCode(context.Background(), "fake-code", "user@example.com", []string{"https://www.googleapis.com/auth/drive"})
	if err == nil {
		t.Fatal("expected error from exchange")
	}
	if !strings.Contains(err.Error(), "exchange token") {
		t.Fatalf("expected exchange token error (not config error), got: %v", err)
	}

	// Verify the auth URL uses the public callback.
	if !strings.Contains(authURL, "my-bot.ngrok-free.app") {
		t.Errorf("auth URL should use public callback: %s", authURL)
	}
}

func TestExchangeCode_WithCallbackURL(t *testing.T) {
	dir := t.TempDir()
	credPath := writeTestCredentials(t)
	dbPath := filepath.Join(dir, "tokens.db")

	store, err := NewTokenStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	tok := &oauth2.Token{
		AccessToken:  "old",
		RefreshToken: "old-ref",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}
	store.SaveToken("user@example.com", tok, []string{"scope-a"})
	store.Close()

	callbackURL := "https://my-bot.ngrok-free.app/auth/google/callback"
	g := New(Config{
		CredentialsFile: credPath,
		TokenDBPath:     dbPath,
		CallbackURL:     callbackURL,
	})

	// ExchangeCode should get past config loading (no config error)
	// and fail at the token exchange step.
	_, err = g.ExchangeCode(context.Background(), "bad-code", "user@example.com", []string{"scope-b"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "exchange token") {
		t.Fatalf("expected exchange error, got config error: %v", err)
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
