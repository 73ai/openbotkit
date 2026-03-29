package client

import (
	"encoding/hex"
	"testing"

	"github.com/zalando/go-keyring"
)

func init() {
	keyring.MockInit()
}

func TestNewSession_GeneratesCSRFToken(t *testing.T) {
	s := NewSession("test-auth-token")
	if s.AuthToken != "test-auth-token" {
		t.Errorf("AuthToken = %q, want test-auth-token", s.AuthToken)
	}
	if s.CSRFToken == "" {
		t.Fatal("CSRFToken should not be empty")
	}
	if len(s.CSRFToken) != 32 {
		t.Errorf("CSRFToken length = %d, want 32 hex chars", len(s.CSRFToken))
	}
	if _, err := hex.DecodeString(s.CSRFToken); err != nil {
		t.Errorf("CSRFToken is not valid hex: %v", err)
	}
}

func TestNewSession_UniqueCSRFTokens(t *testing.T) {
	s1 := NewSession("token1")
	s2 := NewSession("token2")
	if s1.CSRFToken == s2.CSRFToken {
		t.Error("two sessions should have different CSRF tokens")
	}
}

func TestSaveLoadSession(t *testing.T) {
	session := NewSession("my-valid-auth-token-for-testing")
	session.Username = "testuser"

	if err := SaveSession(session); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := LoadSession()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.AuthToken != "my-valid-auth-token-for-testing" {
		t.Errorf("AuthToken = %q, want my-valid-auth-token-for-testing", loaded.AuthToken)
	}
	if loaded.CSRFToken != session.CSRFToken {
		t.Errorf("CSRFToken = %q, want %q", loaded.CSRFToken, session.CSRFToken)
	}
	if loaded.Username != "testuser" {
		t.Errorf("Username = %q, want testuser", loaded.Username)
	}
}

func TestDeleteSession(t *testing.T) {
	session := NewSession("delete-me-long-auth-token-value")
	if err := SaveSession(session); err != nil {
		t.Fatalf("save: %v", err)
	}

	if err := DeleteSession(); err != nil {
		t.Fatalf("delete: %v", err)
	}

	_, err := LoadSession()
	if err == nil {
		t.Fatal("expected error loading deleted session")
	}
}

func TestDeleteSession_NoExistingSession(t *testing.T) {
	if err := DeleteSession(); err != nil {
		t.Fatalf("delete non-existent should not error: %v", err)
	}
}

func TestValidateAuthToken_Empty(t *testing.T) {
	if err := ValidateAuthToken(""); err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestValidateAuthToken_TooShort(t *testing.T) {
	if err := ValidateAuthToken("abc"); err == nil {
		t.Fatal("expected error for short token")
	}
}

func TestValidateAuthToken_Valid(t *testing.T) {
	if err := ValidateAuthToken("a1b2c3d4e5f6g7h8i9j0k1l2"); err != nil {
		t.Fatalf("unexpected error for valid token: %v", err)
	}
}

func TestSaveSession_RejectsEmptyToken(t *testing.T) {
	session := &Session{AuthToken: "", CSRFToken: "x"}
	if err := SaveSession(session); err == nil {
		t.Fatal("expected error saving session with empty token")
	}
}

func TestLoadSession_RegeneratesCSRFIfMissing(t *testing.T) {
	// Save only auth_token directly
	keyring.Set(keyringService, "twitter/auth_token", "some-valid-auth-token-here")

	loaded, err := LoadSession()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.CSRFToken == "" {
		t.Fatal("CSRFToken should be regenerated if missing")
	}
	if len(loaded.CSRFToken) != 32 {
		t.Errorf("CSRFToken length = %d, want 32", len(loaded.CSRFToken))
	}
}
