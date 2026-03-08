package provider

import (
	"fmt"
	"os/exec"
	"strings"

	"golang.org/x/oauth2"
)

// GcloudTokenSource returns an oauth2.TokenSource that uses gcloud CLI
// to obtain access tokens for a specific account.
func GcloudTokenSource(account string) oauth2.TokenSource {
	return &gcloudTokenSource{account: account}
}

type gcloudTokenSource struct{ account string }

func (g *gcloudTokenSource) Token() (*oauth2.Token, error) {
	args := []string{"auth", "print-access-token"}
	if g.account != "" {
		args = append(args, "--account="+g.account)
	}
	out, err := exec.Command("gcloud", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("gcloud auth print-access-token: %w", err)
	}
	return &oauth2.Token{AccessToken: strings.TrimSpace(string(out))}, nil
}
