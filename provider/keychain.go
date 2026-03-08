package provider

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// KeychainLoad retrieves an API key from the macOS Keychain.
// The ref format is "keychain:<service>/<account>", e.g. "keychain:obk/anthropic".
func KeychainLoad(ref string) (string, error) {
	service, account, err := parseKeychainRef(ref)
	if err != nil {
		return "", err
	}

	cmd := exec.Command("security", "find-generic-password",
		"-s", service, "-a", account, "-w")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("keychain lookup failed for %s/%s: %w", service, account, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// KeychainStore saves an API key to the macOS Keychain.
func KeychainStore(ref, value string) error {
	service, account, err := parseKeychainRef(ref)
	if err != nil {
		return err
	}

	// Delete existing entry (ignore errors if it doesn't exist).
	exec.Command("security", "delete-generic-password",
		"-s", service, "-a", account).Run()

	cmd := exec.Command("security", "add-generic-password",
		"-s", service, "-a", account, "-w", value)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("keychain store failed: %w", err)
	}
	return nil
}

// ResolveAPIKey resolves an API key from either a keychain reference
// or an environment variable fallback.
func ResolveAPIKey(ref, envVar string) (string, error) {
	if ref != "" && strings.HasPrefix(ref, "keychain:") {
		key, err := KeychainLoad(ref)
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

func parseKeychainRef(ref string) (service, account string, err error) {
	ref = strings.TrimPrefix(ref, "keychain:")
	parts := strings.SplitN(ref, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid keychain ref %q (want service/account)", ref)
	}
	return parts[0], parts[1], nil
}
