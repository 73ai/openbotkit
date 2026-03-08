//go:build darwin

package provider

import (
	"fmt"
	"os/exec"
	"strings"
)

func credentialLoad(service, account string) (string, error) {
	cmd := exec.Command("security", "find-generic-password",
		"-s", service, "-a", account, "-w")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("keychain lookup failed for %s/%s: %w", service, account, err)
	}
	return strings.TrimSpace(string(out)), nil
}

func credentialStore(service, account, value string) error {
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
