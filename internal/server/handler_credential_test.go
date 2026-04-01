package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/provider"
	"github.com/zalando/go-keyring"
)

func newTestServerWithCreds() *Server {
	s := &Server{
		cfg:        config.Default(),
		credTokens: newCredentialTokenStore(),
	}
	return s
}

func TestCredentialCreate(t *testing.T) {
	s := newTestServerWithCreds()
	body := `{"label":"Test Key","key_ref":"keychain:obk/test"}`
	req := httptest.NewRequest("POST", "/api/credential/request", strings.NewReader(body))
	rec := httptest.NewRecorder()

	s.handleCredentialRequest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["token"] == "" {
		t.Fatal("expected non-empty token")
	}
	if !strings.HasPrefix(resp["url"], "/credential/") {
		t.Fatalf("expected url starting with /credential/, got %q", resp["url"])
	}
}

func TestCredentialCreate_RequiresAuth(t *testing.T) {
	s := newTestServerWithCreds()
	s.cfg.Auth = &config.AuthConfig{Username: "admin", Password: "secret"}

	body := `{"label":"Test Key","key_ref":"keychain:obk/test"}`
	req := httptest.NewRequest("POST", "/api/credential/request", strings.NewReader(body))
	rec := httptest.NewRecorder()

	// Call through the auth middleware, not the handler directly.
	handler := s.basicAuth(http.HandlerFunc(s.handleCredentialRequest))
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without auth, got %d", rec.Code)
	}
}

func TestCredentialCreate_MissingFields(t *testing.T) {
	s := newTestServerWithCreds()
	body := `{"label":"Test Key"}`
	req := httptest.NewRequest("POST", "/api/credential/request", strings.NewReader(body))
	rec := httptest.NewRecorder()

	s.handleCredentialRequest(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCredentialFormRender(t *testing.T) {
	s := newTestServerWithCreds()
	token, err := s.credTokens.create("Anthropic API Key", "keychain:obk/anthropic")
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	req := httptest.NewRequest("GET", "/credential/"+token, nil)
	req.SetPathValue("token", token)
	rec := httptest.NewRecorder()

	s.handleCredentialForm(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Anthropic API Key") {
		t.Error("form should contain the label")
	}
	if !strings.Contains(body, `type="password"`) {
		t.Error("form should have a password input")
	}
}

func TestCredentialSubmit(t *testing.T) {
	keyring.MockInit()

	s := newTestServerWithCreds()
	token, err := s.credTokens.create("Test Key", "keychain:obk/test-handler")
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	form := url.Values{"value": {"sk-test-12345"}}
	req := httptest.NewRequest("POST", "/credential/"+token, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("token", token)
	rec := httptest.NewRecorder()

	s.handleCredentialSubmit(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if !strings.Contains(body, "saved successfully") {
		t.Error("success page should say 'saved successfully'")
	}
}

func TestCredentialSubmit_StoresInKeyring(t *testing.T) {
	keyring.MockInit()

	s := newTestServerWithCreds()
	token, _ := s.credTokens.create("Test Key", "keychain:obk/test-store")

	form := url.Values{"value": {"my-secret-key"}}
	req := httptest.NewRequest("POST", "/credential/"+token, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("token", token)
	rec := httptest.NewRecorder()

	s.handleCredentialSubmit(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify the credential was actually stored.
	val, err := provider.LoadCredential("keychain:obk/test-store")
	if err != nil {
		t.Fatalf("LoadCredential: %v", err)
	}
	if val != "my-secret-key" {
		t.Errorf("stored value = %q, want %q", val, "my-secret-key")
	}
}

func TestCredentialSubmit_InvalidatesToken(t *testing.T) {
	keyring.MockInit()

	s := newTestServerWithCreds()
	token, _ := s.credTokens.create("Test Key", "keychain:obk/test-invalidate")

	// Submit the credential.
	form := url.Values{"value": {"sk-test"}}
	req := httptest.NewRequest("POST", "/credential/"+token, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("token", token)
	rec := httptest.NewRecorder()
	s.handleCredentialSubmit(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("first submit expected 200, got %d", rec.Code)
	}

	// Second access to the same token should fail.
	req2 := httptest.NewRequest("GET", "/credential/"+token, nil)
	req2.SetPathValue("token", token)
	rec2 := httptest.NewRecorder()
	s.handleCredentialForm(rec2, req2)

	if rec2.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for used token, got %d", rec2.Code)
	}
}

func TestCredentialExpiry(t *testing.T) {
	s := newTestServerWithCreds()
	token, _ := s.credTokens.create("Test Key", "keychain:obk/test-expiry")

	// Manually set expiry in the past.
	s.credTokens.mu.Lock()
	s.credTokens.tokens[token].ExpiresAt = time.Now().Add(-1 * time.Minute)
	s.credTokens.mu.Unlock()

	req := httptest.NewRequest("GET", "/credential/"+token, nil)
	req.SetPathValue("token", token)
	rec := httptest.NewRecorder()

	s.handleCredentialForm(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for expired token, got %d", rec.Code)
	}
}

func TestCredentialInvalidToken(t *testing.T) {
	s := newTestServerWithCreds()

	req := httptest.NewRequest("GET", "/credential/bogus-token", nil)
	req.SetPathValue("token", "bogus-token")
	rec := httptest.NewRecorder()

	s.handleCredentialForm(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for invalid token, got %d", rec.Code)
	}
}

func TestCredentialCreate_CleansExpiredTokens(t *testing.T) {
	s := newTestServerWithCreds()

	// Create a token and expire it manually.
	old, _ := s.credTokens.create("Old Key", "keychain:obk/old")
	s.credTokens.mu.Lock()
	s.credTokens.tokens[old].ExpiresAt = time.Now().Add(-1 * time.Minute)
	s.credTokens.mu.Unlock()

	// Creating a new token should sweep the expired one.
	s.credTokens.create("New Key", "keychain:obk/new")

	s.credTokens.mu.Lock()
	_, exists := s.credTokens.tokens[old]
	s.credTokens.mu.Unlock()

	if exists {
		t.Error("expired token should have been cleaned up on create")
	}
}

func TestCredentialSubmit_EmptyValue(t *testing.T) {
	s := newTestServerWithCreds()
	token, _ := s.credTokens.create("Test Key", "keychain:obk/test-empty")

	form := url.Values{"value": {""}}
	req := httptest.NewRequest("POST", "/credential/"+token, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("token", token)
	rec := httptest.NewRecorder()

	s.handleCredentialSubmit(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty value, got %d", rec.Code)
	}
}
