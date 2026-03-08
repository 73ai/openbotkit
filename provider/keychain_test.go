package provider

import (
	"testing"
)

func TestParseKeychainRef(t *testing.T) {
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
			service, account, err := parseKeychainRef(tt.ref)
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
