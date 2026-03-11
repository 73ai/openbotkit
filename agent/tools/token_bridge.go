package tools

import (
	"context"

	"github.com/priyanshujain/openbotkit/oauth/google"
)

// TokenBridge injects obk's OAuth token into gws command environment.
type TokenBridge struct {
	google  *google.Google
	account string
}

func NewTokenBridge(g *google.Google, account string) *TokenBridge {
	return &TokenBridge{google: g, account: account}
}

// Env returns environment variables with the access token for gws.
func (tb *TokenBridge) Env(ctx context.Context) ([]string, error) {
	token, err := tb.google.AccessToken(ctx, tb.account)
	if err != nil {
		return nil, err
	}
	return []string{"GOOGLE_WORKSPACE_CLI_TOKEN=" + token}, nil
}
