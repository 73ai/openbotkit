package provider

import (
	"fmt"
	"os"
	"strings"

	"github.com/zalando/go-keyring"
)

func LoadCredential(ref string) (string, error) {
	service, account, err := parseCredentialRef(ref)
	if err != nil {
		return "", err
	}
	val, err := keyring.Get(service, account)
	if err != nil {
		return "", fmt.Errorf("load credential %s/%s: %w", service, account, err)
	}
	return val, nil
}

func StoreCredential(ref, value string) error {
	service, account, err := parseCredentialRef(ref)
	if err != nil {
		return err
	}
	if err := keyring.Set(service, account, value); err != nil {
		return fmt.Errorf("store credential %s/%s: %w", service, account, err)
	}
	return nil
}

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

func parseCredentialRef(ref string) (service, account string, err error) {
	ref = strings.TrimPrefix(ref, "keychain:")
	parts := strings.SplitN(ref, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid credential ref %q (want service/account)", ref)
	}
	return parts[0], parts[1], nil
}
