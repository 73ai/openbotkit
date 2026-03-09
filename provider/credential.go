package provider

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zalando/go-keyring"
)

// LoadCredential retrieves an API key from the platform credential store.
// The ref format is "keychain:<service>/<account>", e.g. "keychain:obk/anthropic".
func LoadCredential(ref string) (string, error) {
	service, account, err := parseCredentialRef(ref)
	if err != nil {
		return "", err
	}
	return credentialLoad(service, account)
}

// StoreCredential saves an API key to the platform credential store.
func StoreCredential(ref, value string) error {
	service, account, err := parseCredentialRef(ref)
	if err != nil {
		return err
	}
	return credentialStore(service, account, value)
}

// ResolveAPIKey resolves an API key from either a credential store reference
// or an environment variable fallback.
func ResolveAPIKey(ref, envVar string) (string, error) {
	if ref != "" && strings.HasPrefix(ref, "keychain:") {
		key, err := LoadCredential(ref)
		if err == nil && key != "" {
			return key, nil
		}
	}

	if envVar != "" {
		if key := os.Getenv(envVar); key != "" {
			return key, nil
		}
	}

	return "", fmt.Errorf("no API key found (ref=%q, env=%q)", ref, envVar)
}

// keyringUnavailable returns true for errors that indicate the OS keyring
// daemon is not available (headless server, Docker, unsupported platform).
// Operational errors like ErrSetDataTooBig are not considered unavailable.
func keyringUnavailable(err error) bool {
	return !errors.Is(err, keyring.ErrSetDataTooBig)
}

// credentialLoad tries the OS keyring first, falls back to file-based storage
// when the keyring is unavailable (headless/Docker). Operational keyring errors
// are propagated.
func credentialLoad(service, account string) (string, error) {
	val, err := keyring.Get(service, account)
	if err == nil {
		return val, nil
	}
	if !keyringUnavailable(err) {
		return "", fmt.Errorf("keyring: %w", err)
	}
	return loadFromFile(service, account)
}

// credentialStore tries the OS keyring first, falls back to file-based storage
// when the keyring is unavailable (headless/Docker). Operational keyring errors
// are propagated.
func credentialStore(service, account, value string) error {
	err := keyring.Set(service, account, value)
	if err == nil {
		return nil
	}
	if !keyringUnavailable(err) {
		return fmt.Errorf("keyring: %w", err)
	}
	return storeToFile(service, account, value)
}

func parseCredentialRef(ref string) (service, account string, err error) {
	ref = strings.TrimPrefix(ref, "keychain:")
	parts := strings.SplitN(ref, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid credential ref %q (want service/account)", ref)
	}
	return parts[0], parts[1], nil
}

func secretsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, ".obk", "secrets"), nil
}

func secretPath(service, account string) (string, error) {
	dir, err := secretsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, service+"-"+account), nil
}

// loadFromFile reads a credential from the file-based store.
func loadFromFile(service, account string) (string, error) {
	path, err := secretPath(service, account)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("credential lookup failed for %s/%s: %w", service, account, err)
	}
	return strings.TrimSpace(string(data)), nil
}

// storeToFile writes a credential to the file-based store with 0600 permissions.
func storeToFile(service, account, value string) error {
	dir, err := secretsDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create secrets dir: %w", err)
	}
	path, err := secretPath(service, account)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(value), 0600); err != nil {
		return fmt.Errorf("store credential: %w", err)
	}
	return nil
}
