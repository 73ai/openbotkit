package desktop

import (
	"context"
	"fmt"

	"github.com/73ai/openbotkit/source/slack"
)

type Credentials struct {
	Token    string
	Cookie   string
	TeamID   string
	TeamName string
}

type authTestFunc func(token, cookie string) (teamID, teamName string, err error)

// Extract returns credentials for every workspace signed in to Slack Desktop.
func Extract() ([]*Credentials, error) {
	tokens, err := ExtractTokens()
	if err != nil {
		return nil, fmt.Errorf("extract token: %w", err)
	}

	cookie, err := ExtractCookie()
	if err != nil {
		return nil, fmt.Errorf("extract cookie: %w", err)
	}

	return validateTokens(tokens, cookie, func(token, cookie string) (string, string, error) {
		teamID, teamName, _, err := slack.NewClient(token, cookie).AuthTest(context.Background())
		return teamID, teamName, err
	})
}

// validateTokens keeps tokens that pass auth.test, one per team. Stale tokens
// from signed-out workspaces linger in storage and are skipped.
func validateTokens(tokens []string, cookie string, authTest authTestFunc) ([]*Credentials, error) {
	var creds []*Credentials
	seenTeams := make(map[string]bool)
	var lastErr error
	for _, token := range tokens {
		teamID, teamName, err := authTest(token, cookie)
		if err != nil {
			lastErr = err
			continue
		}
		if seenTeams[teamID] {
			continue
		}
		seenTeams[teamID] = true
		creds = append(creds, &Credentials{
			Token:    token,
			Cookie:   cookie,
			TeamID:   teamID,
			TeamName: teamName,
		})
	}
	if len(creds) == 0 {
		return nil, fmt.Errorf("validate credentials: %w", lastErr)
	}
	return creds, nil
}
