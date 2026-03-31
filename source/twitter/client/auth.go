package client

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/zalando/go-keyring"
)

const keyringService = "obk"

type Session struct {
	AuthToken string
	CSRFToken string
	Username  string
}

func NewSession(authToken string) *Session {
	return &Session{
		AuthToken: authToken,
		CSRFToken: generateCSRFToken(),
	}
}

func NewSessionWithCSRF(authToken, csrfToken string) *Session {
	if csrfToken == "" {
		csrfToken = generateCSRFToken()
	}
	return &Session{
		AuthToken: authToken,
		CSRFToken: csrfToken,
	}
}

func ValidateAuthToken(token string) error {
	if token == "" {
		return fmt.Errorf("auth_token is empty")
	}
	if len(token) < 20 {
		return fmt.Errorf("auth_token appears too short (got %d chars)", len(token))
	}
	return nil
}

func SaveSession(session *Session) error {
	if err := ValidateAuthToken(session.AuthToken); err != nil {
		return err
	}
	if err := keyring.Set(keyringService, "twitter/auth_token", session.AuthToken); err != nil {
		return fmt.Errorf("save auth_token: %w", err)
	}
	if err := keyring.Set(keyringService, "twitter/csrf_token", session.CSRFToken); err != nil {
		return fmt.Errorf("save csrf_token: %w", err)
	}
	if session.Username != "" {
		if err := keyring.Set(keyringService, "twitter/username", session.Username); err != nil {
			return fmt.Errorf("save username: %w", err)
		}
	}
	return nil
}

func LoadSession() (*Session, error) {
	authToken, err := keyring.Get(keyringService, "twitter/auth_token")
	if err != nil {
		return nil, fmt.Errorf("load auth_token: %w", err)
	}
	csrfToken, _ := keyring.Get(keyringService, "twitter/csrf_token")
	if csrfToken == "" {
		csrfToken = generateCSRFToken()
	}
	username, _ := keyring.Get(keyringService, "twitter/username")
	return &Session{
		AuthToken: authToken,
		CSRFToken: csrfToken,
		Username:  username,
	}, nil
}

func DeleteSession() error {
	var errs []error
	for _, key := range []string{"twitter/auth_token", "twitter/csrf_token", "twitter/username"} {
		if err := keyring.Delete(keyringService, key); err != nil && err != keyring.ErrNotFound {
			errs = append(errs, fmt.Errorf("delete %s: %w", key, err))
		}
	}
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// ValidateSession makes a lightweight API call to verify the session
// tokens actually work. Returns nil if the session is valid.
func ValidateSession(ctx context.Context, session *Session) error {
	c, err := NewClient(session, "")
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}
	// Fetch 1 tweet from the timeline as a health check.
	if _, err := c.HomeLatestTimeline(ctx, 1, ""); err != nil {
		return err
	}
	return nil
}

func generateCSRFToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand failed: %v", err))
	}
	return hex.EncodeToString(b)
}
