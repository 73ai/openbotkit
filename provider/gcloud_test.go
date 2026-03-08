package provider

import (
	"os/exec"
	"testing"
)

func TestGcloudTokenSource_Live(t *testing.T) {
	if _, err := exec.LookPath("gcloud"); err != nil {
		t.Skip("gcloud not installed")
	}

	// Use empty account to use the default active account.
	ts := GcloudTokenSource("")
	tok, err := ts.Token()
	if err != nil {
		t.Fatalf("Token: %v", err)
	}
	if tok.AccessToken == "" {
		t.Error("got empty access token")
	}
	if len(tok.AccessToken) < 20 {
		t.Errorf("access token looks too short: %q", tok.AccessToken)
	}
}
