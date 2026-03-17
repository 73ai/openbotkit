package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"github.com/73ai/openbotkit/config"
	google "github.com/73ai/openbotkit/oauth/google"
)

func TestHandleGoogleAuthCallback_MissingCode(t *testing.T) {
	s := &Server{cfg: &config.Config{}}
	req := httptest.NewRequest("GET", "/auth/google/callback?state=abc", nil)
	rec := httptest.NewRecorder()
	s.handleGoogleAuthCallback(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandleGoogleAuthCallback_MissingState(t *testing.T) {
	s := &Server{cfg: &config.Config{}}
	req := httptest.NewRequest("GET", "/auth/google/callback?code=abc", nil)
	rec := httptest.NewRecorder()
	s.handleGoogleAuthCallback(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestResolveAccount_ReturnsFirstAccount(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "tokens.db")

	store, _ := google.NewTokenStore(dbPath)
	tok := &oauth2.Token{
		AccessToken:  "tok",
		RefreshToken: "ref",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}
	store.SaveToken("alice@test.com", tok, []string{"openid"})
	store.Close()

	credPath := filepath.Join(dir, "credentials.json")
	os.WriteFile(credPath, []byte(testCredsJSON), 0600)

	s := &Server{
		cfg: &config.Config{},
		google: google.New(google.Config{
			CredentialsFile: credPath,
			TokenDBPath:     dbPath,
		}),
	}
	s.ctx = context.Background()

	account := s.resolveAccount()
	if account != "alice@test.com" {
		t.Errorf("resolveAccount = %q, want alice@test.com", account)
	}
}

func TestResolveAccount_NoAccounts(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "tokens.db")
	credPath := filepath.Join(dir, "credentials.json")
	os.WriteFile(credPath, []byte(testCredsJSON), 0600)

	// Create empty token store.
	store, _ := google.NewTokenStore(dbPath)
	store.Close()

	s := &Server{
		cfg: &config.Config{},
		google: google.New(google.Config{
			CredentialsFile: credPath,
			TokenDBPath:     dbPath,
		}),
	}
	s.ctx = context.Background()

	account := s.resolveAccount()
	if account != "" {
		t.Errorf("resolveAccount = %q, want empty", account)
	}
}

func TestHandleGoogleAuthCallback_EmptyAccountAttemptsExchange(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "tokens.db")
	credPath := filepath.Join(dir, "credentials.json")
	os.WriteFile(credPath, []byte(testCredsJSON), 0600)

	store, _ := google.NewTokenStore(dbPath)
	store.Close()

	waiter := google.NewScopeWaiter()
	s := &Server{
		cfg:         &config.Config{},
		scopeWaiter: waiter,
		google: google.New(google.Config{
			CredentialsFile: credPath,
			TokenDBPath:     dbPath,
		}),
	}
	s.ctx = context.Background()

	// Register a waiter with scopes but empty account so Lookup succeeds.
	// The handler should still attempt exchange (not reject at 400).
	// The exchange fails because the code is fake, yielding 500.
	go waiter.Wait("empty-account-state", 5*time.Second, []string{"openid"}, "")
	time.Sleep(20 * time.Millisecond)

	req := httptest.NewRequest("GET", "/auth/google/callback?code=abc&state=empty-account-state", nil)
	rec := httptest.NewRecorder()
	s.handleGoogleAuthCallback(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500 (exchange failure, not 400 rejection)", rec.Code)
	}
}

const testCredsJSON = `{
	"installed": {
		"client_id": "test.apps.googleusercontent.com",
		"client_secret": "secret",
		"auth_uri": "https://accounts.google.com/o/oauth2/auth",
		"token_uri": "https://oauth2.googleapis.com/token",
		"redirect_uris": ["http://localhost"]
	}
}`

func TestHandleGoogleAuthCallback_ExchangeFailureSignalsError(t *testing.T) {
	dir := t.TempDir()
	credPath := filepath.Join(dir, "credentials.json")
	os.WriteFile(credPath, []byte(testCredsJSON), 0600)

	dbPath := filepath.Join(dir, "tokens.db")
	store, _ := google.NewTokenStore(dbPath)
	tok := &oauth2.Token{
		AccessToken:  "tok",
		RefreshToken: "ref",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}
	store.SaveToken("user@test.com", tok, []string{"openid"})
	store.Close()

	waiter := google.NewScopeWaiter()
	g := google.New(google.Config{
		CredentialsFile: credPath,
		TokenDBPath:     dbPath,
	})

	s := &Server{
		cfg:         &config.Config{},
		scopeWaiter: waiter,
		google:      g,
	}

	// Start a waiter that should receive the error signal.
	signaled := make(chan error, 1)
	go func() {
		signaled <- waiter.Wait("test-state", 5*time.Second, []string{"calendar"}, "user@test.com")
	}()
	time.Sleep(20 * time.Millisecond)

	req := httptest.NewRequest("GET", "/auth/google/callback?code=bad-code&state=test-state", nil)
	rec := httptest.NewRecorder()
	s.handleGoogleAuthCallback(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}

	// Verify the waiter was signaled with an error (not left hanging).
	select {
	case err := <-signaled:
		if err == nil {
			t.Fatal("expected error from waiter, got nil")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("waiter was not signaled — tool would hang until timeout")
	}
}
