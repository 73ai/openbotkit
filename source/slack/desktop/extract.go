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

func Extract() (*Credentials, error) {
	token, err := ExtractToken()
	if err != nil {
		return nil, fmt.Errorf("extract token: %w", err)
	}

	cookie, err := ExtractCookie()
	if err != nil {
		return nil, fmt.Errorf("extract cookie: %w", err)
	}

	client := slack.NewClient(token, cookie)
	teamID, teamName, _, err := client.AuthTest(context.Background())
	if err != nil {
		return nil, fmt.Errorf("validate credentials: %w", err)
	}

	return &Credentials{
		Token:    token,
		Cookie:   cookie,
		TeamID:   teamID,
		TeamName: teamName,
	}, nil
}
