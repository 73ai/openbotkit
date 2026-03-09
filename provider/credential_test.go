package provider

import (
	"testing"

	"github.com/zalando/go-keyring"
)

func TestParseCredentialRef(t *testing.T) {
	tests := []struct {
		ref     string
		service string
		account string
		wantErr bool
	}{
		{"keychain:obk/anthropic", "obk", "anthropic", false},
		{"keychain:my-service/my-account", "my-service", "my-account", false},
		{"obk/anthropic", "obk", "anthropic", false}, // prefix already stripped
		{"no-slash", "", "", true},
		{"", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			service, account, err := parseCredentialRef(tt.ref)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tt.wantErr)
			}
			if service != tt.service {
				t.Errorf("service = %q, want %q", service, tt.service)
			}
			if account != tt.account {
				t.Errorf("account = %q, want %q", account, tt.account)
			}
		})
	}
}

func TestStoreLoadCredential(t *testing.T) {
	keyring.MockInit()

	ref := "keychain:obk/test-provider"
	if err := StoreCredential(ref, "my-secret"); err != nil {
		t.Fatalf("StoreCredential: %v", err)
	}

	val, err := LoadCredential(ref)
	if err != nil {
		t.Fatalf("LoadCredential: %v", err)
	}
	if val != "my-secret" {
		t.Errorf("got %q, want %q", val, "my-secret")
	}
}

func TestLoadCredential_NotFound(t *testing.T) {
	keyring.MockInit()

	_, err := LoadCredential("keychain:obk/nonexistent")
	if err == nil {
		t.Fatal("expected error for missing credential")
	}
}

func TestStoreCredential_KeyringError(t *testing.T) {
	keyring.MockInitWithError(keyring.ErrSetDataTooBig)

	err := StoreCredential("keychain:obk/test", "value")
	if err == nil {
		t.Fatal("expected error when keyring fails")
	}
}

func TestLoadCredential_KeyringError(t *testing.T) {
	keyring.MockInitWithError(keyring.ErrUnsupportedPlatform)

	_, err := LoadCredential("keychain:obk/test")
	if err == nil {
		t.Fatal("expected error when keyring unavailable")
	}
}

func TestResolveAPIKey_EnvFallback(t *testing.T) {
	t.Setenv("TEST_API_KEY_XYZ", "test-value-123")

	key, err := ResolveAPIKey("", "TEST_API_KEY_XYZ")
	if err != nil {
		t.Fatalf("ResolveAPIKey: %v", err)
	}
	if key != "test-value-123" {
		t.Errorf("key = %q, want %q", key, "test-value-123")
	}
}

func TestResolveAPIKey_NoKeyFound(t *testing.T) {
	_, err := ResolveAPIKey("", "NONEXISTENT_KEY_VAR_12345")
	if err == nil {
		t.Fatal("expected error when no key is available")
	}
}

func TestResolveAPIKey_EnvOverridesEmptyRef(t *testing.T) {
	t.Setenv("TEST_RESOLVE_KEY", "from-env")

	// Non-keychain ref is ignored, falls through to env.
	key, err := ResolveAPIKey("not-a-keychain-ref", "TEST_RESOLVE_KEY")
	if err != nil {
		t.Fatalf("ResolveAPIKey: %v", err)
	}
	if key != "from-env" {
		t.Errorf("key = %q, want %q", key, "from-env")
	}
}

func TestResolveAPIKey_KeyringThenEnv(t *testing.T) {
	keyring.MockInit()
	t.Setenv("TEST_RESOLVE_BOTH", "from-env")

	// Store in keyring — should prefer keyring over env.
	if err := StoreCredential("keychain:obk/resolve-test", "from-keyring"); err != nil {
		t.Fatalf("StoreCredential: %v", err)
	}

	key, err := ResolveAPIKey("keychain:obk/resolve-test", "TEST_RESOLVE_BOTH")
	if err != nil {
		t.Fatalf("ResolveAPIKey: %v", err)
	}
	if key != "from-keyring" {
		t.Errorf("got %q, want %q", key, "from-keyring")
	}
}
