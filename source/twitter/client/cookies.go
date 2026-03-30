package client

import (
	"fmt"

	"github.com/73ai/openbotkit/internal/browser/cookies"
)

func ExtractSessionFromBrowser() (*Session, string, error) {
	result, err := cookies.ExtractTwitterCookies()
	if err != nil {
		return nil, "", fmt.Errorf("extract browser cookies: %w", err)
	}

	if err := ValidateAuthToken(result.AuthToken); err != nil {
		return nil, "", fmt.Errorf("invalid auth_token from %s: %w", result.Browser, err)
	}

	session := NewSessionWithCSRF(result.AuthToken, result.CSRFToken)
	return session, result.Browser, nil
}
