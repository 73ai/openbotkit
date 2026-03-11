package slack

import (
	"fmt"
	"strings"

	"github.com/zalando/go-keyring"
)

const keyringService = "obk"

func tokenAccount(workspace string) string  { return fmt.Sprintf("slack/%s/token", workspace) }
func cookieAccount(workspace string) string { return fmt.Sprintf("slack/%s/cookie", workspace) }

type Credentials struct {
	Token  string
	Cookie string
}

func SaveCredentials(workspace, token, cookie string) error {
	if err := keyring.Set(keyringService, tokenAccount(workspace), token); err != nil {
		return fmt.Errorf("save token: %w", err)
	}
	if cookie != "" {
		if err := keyring.Set(keyringService, cookieAccount(workspace), cookie); err != nil {
			return fmt.Errorf("save cookie: %w", err)
		}
	}
	return nil
}

func LoadCredentials(workspace string) (*Credentials, error) {
	token, err := keyring.Get(keyringService, tokenAccount(workspace))
	if err != nil {
		return nil, fmt.Errorf("load token: %w", err)
	}
	cookie, _ := keyring.Get(keyringService, cookieAccount(workspace))
	return &Credentials{Token: token, Cookie: cookie}, nil
}

func DeleteCredentials(workspace string) error {
	_ = keyring.Delete(keyringService, tokenAccount(workspace))
	_ = keyring.Delete(keyringService, cookieAccount(workspace))
	return nil
}

func ListWorkspaces() ([]string, error) {
	// go-keyring doesn't support listing, so we rely on config.
	// This is a helper that callers fill from config.Slack.Workspaces keys.
	return nil, fmt.Errorf("use config.Slack.Workspaces to list workspaces")
}

func ValidateToken(token string) error {
	if token == "" {
		return fmt.Errorf("token is empty")
	}
	validPrefixes := []string{"xoxp-", "xoxb-", "xoxc-"}
	for _, p := range validPrefixes {
		if strings.HasPrefix(token, p) {
			return nil
		}
	}
	return fmt.Errorf("token must start with xoxp-, xoxb-, or xoxc-")
}
