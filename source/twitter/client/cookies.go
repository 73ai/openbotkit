package client

import (
	"fmt"

	"github.com/73ai/openbotkit/internal/browser/cookies"
)

// ExtractSessionFromBrowser tries all available browsers in order.
func ExtractSessionFromBrowser() (*Session, string, error) {
	result, err := cookies.ExtractTwitterCookies()
	if err != nil {
		return nil, "", fmt.Errorf("extract browser cookies: %w", err)
	}
	return sessionFromResult(result)
}

// ExtractSessionFromBrowserByName extracts cookies from a specific browser.
func ExtractSessionFromBrowserByName(browser string) (*Session, string, error) {
	result, err := cookies.ExtractTwitterCookiesFromBrowser(browser)
	if err != nil {
		return nil, "", err
	}
	return sessionFromResult(result)
}

func sessionFromResult(result *cookies.Result) (*Session, string, error) {
	if err := ValidateAuthToken(result.AuthToken); err != nil {
		return nil, "", fmt.Errorf("invalid auth_token from %s: %w", result.Browser, err)
	}
	session := NewSessionWithCSRF(result.AuthToken, result.CSRFToken)
	return session, result.Browser, nil
}
