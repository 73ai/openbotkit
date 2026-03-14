package google

import (
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

// fakeTokenSource returns a new token on every call.
type fakeTokenSource struct {
	tok *oauth2.Token
}

func (f *fakeTokenSource) Token() (*oauth2.Token, error) {
	return f.tok, nil
}

// testTokenStoreFile creates a token store backed by a temp file (not :memory:)
// so that dbTokenSource can open its own connection for writes.
func testTokenStoreFile(t *testing.T) (*TokenStore, string) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "tokens.db")
	ts, err := NewTokenStore(dbPath)
	if err != nil {
		t.Fatalf("open token store: %v", err)
	}
	t.Cleanup(func() { ts.Close() })
	return ts, dbPath
}

func TestDBTokenSourceReturnsValidToken(t *testing.T) {
	_, dbPath := testTokenStoreFile(t)

	initial := &oauth2.Token{
		AccessToken:  "valid-access",
		RefreshToken: "r1",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}
	ts, _ := NewTokenStore(dbPath)
	ts.SaveToken("user@gmail.com", initial, []string{"scope1"})
	ts.Close()

	src := newDBTokenSource("user@gmail.com", dbPath, nil, initial)

	got, err := src.Token()
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	if got.AccessToken != "valid-access" {
		t.Errorf("expected valid-access, got %q", got.AccessToken)
	}
}

func TestDBTokenSourceRefreshesExpiredToken(t *testing.T) {
	ts, dbPath := testTokenStoreFile(t)

	expired := &oauth2.Token{
		AccessToken:  "expired",
		RefreshToken: "r1",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(-time.Hour),
	}
	ts.SaveToken("user@gmail.com", expired, []string{"scope1"})

	refreshed := &oauth2.Token{
		AccessToken:  "refreshed-access",
		RefreshToken: "r1",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}
	fake := &fakeTokenSource{tok: refreshed}
	src := newDBTokenSource("user@gmail.com", dbPath, fake, expired)

	got, err := src.Token()
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	if got.AccessToken != "refreshed-access" {
		t.Errorf("expected refreshed-access, got %q", got.AccessToken)
	}

	// Verify it was persisted by opening a fresh store.
	verifyStore, err := NewTokenStore(dbPath)
	if err != nil {
		t.Fatalf("open verify store: %v", err)
	}
	defer verifyStore.Close()

	loaded, _, err := verifyStore.LoadToken("user@gmail.com")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.AccessToken != "refreshed-access" {
		t.Errorf("persisted access token: got %q, want %q", loaded.AccessToken, "refreshed-access")
	}
}

func TestDBTokenSourcePersistsRotatedRefreshToken(t *testing.T) {
	ts, dbPath := testTokenStoreFile(t)

	initial := &oauth2.Token{
		AccessToken:  "a1",
		RefreshToken: "old-refresh",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(-time.Hour),
	}
	ts.SaveToken("user@gmail.com", initial, []string{"scope1"})

	rotated := &oauth2.Token{
		AccessToken:  "a2",
		RefreshToken: "new-refresh",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}
	fake := &fakeTokenSource{tok: rotated}
	src := newDBTokenSource("user@gmail.com", dbPath, fake, initial)

	_, err := src.Token()
	if err != nil {
		t.Fatalf("token: %v", err)
	}

	// Verify via fresh store.
	verifyStore, err := NewTokenStore(dbPath)
	if err != nil {
		t.Fatalf("open verify store: %v", err)
	}
	defer verifyStore.Close()

	loaded, _, err := verifyStore.LoadToken("user@gmail.com")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.RefreshToken != "new-refresh" {
		t.Errorf("refresh token: got %q, want %q", loaded.RefreshToken, "new-refresh")
	}
}

func TestDBTokenSourceWorksAfterOriginalStoreClosed(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "tokens.db")

	// Simulate Client(): open store, seed data, create dbTokenSource, close store.
	store, err := NewTokenStore(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	expired := &oauth2.Token{
		AccessToken:  "expired",
		RefreshToken: "r1",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(-time.Hour),
	}
	store.SaveToken("user@gmail.com", expired, []string{"scope1"})

	refreshed := &oauth2.Token{
		AccessToken:  "new-access",
		RefreshToken: "r1",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}
	fake := &fakeTokenSource{tok: refreshed}
	src := newDBTokenSource("user@gmail.com", dbPath, fake, expired)

	// Close the original store — this is exactly what Client() does via defer.
	store.Close()

	// Token() must still work: opens its own DB connection for the save.
	got, err := src.Token()
	if err != nil {
		t.Fatalf("token after store closed: %v", err)
	}
	if got.AccessToken != "new-access" {
		t.Errorf("expected new-access, got %q", got.AccessToken)
	}
}
