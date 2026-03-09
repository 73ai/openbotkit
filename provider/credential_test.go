package provider

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
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

// setTestHome overrides the home directory to dir for the duration of the test.
// On Windows os.UserHomeDir reads USERPROFILE; on Unix it reads HOME.
func setTestHome(t *testing.T, dir string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", dir)
	} else {
		t.Setenv("HOME", dir)
	}
}

func TestFileCredentialStoreLoad(t *testing.T) {
	dir := t.TempDir()
	setTestHome(t, dir)

	err := storeToFile("obk", "test-provider", "secret-key-123")
	if err != nil {
		t.Fatalf("storeToFile: %v", err)
	}

	// Verify file exists with correct permissions (Unix only).
	path := filepath.Join(dir, ".obk", "secrets", "obk-test-provider")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat secret file: %v", err)
	}
	if runtime.GOOS != "windows" {
		if perm := info.Mode().Perm(); perm != 0600 {
			t.Errorf("file permissions = %o, want 0600", perm)
		}
	}

	// Load it back.
	val, err := loadFromFile("obk", "test-provider")
	if err != nil {
		t.Fatalf("loadFromFile: %v", err)
	}
	if val != "secret-key-123" {
		t.Errorf("loaded value = %q, want %q", val, "secret-key-123")
	}
}

func TestFileCredentialLoad_NotFound(t *testing.T) {
	dir := t.TempDir()
	setTestHome(t, dir)

	_, err := loadFromFile("obk", "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing credential")
	}
}

func TestCredentialStore_KeyringSuccess(t *testing.T) {
	keyring.MockInit()

	err := credentialStore("obk", "test-kr", "keyring-secret")
	if err != nil {
		t.Fatalf("credentialStore: %v", err)
	}

	val, err := credentialLoad("obk", "test-kr")
	if err != nil {
		t.Fatalf("credentialLoad: %v", err)
	}
	if val != "keyring-secret" {
		t.Errorf("got %q, want %q", val, "keyring-secret")
	}
}

func TestCredentialStore_FallbackToFile(t *testing.T) {
	keyring.MockInitWithError(errors.New("no keyring"))
	dir := t.TempDir()
	setTestHome(t, dir)

	err := credentialStore("obk", "test-fb", "file-secret")
	if err != nil {
		t.Fatalf("credentialStore: %v", err)
	}

	// Verify it was written to file, not keyring.
	path := filepath.Join(dir, ".obk", "secrets", "obk-test-fb")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file at %s: %v", path, err)
	}

	val, err := credentialLoad("obk", "test-fb")
	if err != nil {
		t.Fatalf("credentialLoad: %v", err)
	}
	if val != "file-secret" {
		t.Errorf("got %q, want %q", val, "file-secret")
	}
}

func TestStoreLoadCredential_KeyringRoundTrip(t *testing.T) {
	keyring.MockInit()

	ref := "keychain:obk/test-exported"
	if err := StoreCredential(ref, "exported-secret"); err != nil {
		t.Fatalf("StoreCredential: %v", err)
	}

	val, err := LoadCredential(ref)
	if err != nil {
		t.Fatalf("LoadCredential: %v", err)
	}
	if val != "exported-secret" {
		t.Errorf("got %q, want %q", val, "exported-secret")
	}
}

func TestCredentialStore_PropagatesOperationalError(t *testing.T) {
	keyring.MockInitWithError(keyring.ErrSetDataTooBig)
	dir := t.TempDir()
	setTestHome(t, dir)

	err := credentialStore("obk", "test-big", "some-value")
	if err == nil {
		t.Fatal("expected error for ErrSetDataTooBig")
	}
	if !errors.Is(err, keyring.ErrSetDataTooBig) {
		t.Errorf("expected ErrSetDataTooBig, got: %v", err)
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
