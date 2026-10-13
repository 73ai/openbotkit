package desktop

import (
	"errors"
	"testing"
)

func TestCredentials_Fields(t *testing.T) {
	c := &Credentials{
		Token:    "xoxc-abc",
		Cookie:   "xoxd-def",
		TeamID:   "T123",
		TeamName: "TestTeam",
	}
	if c.Token != "xoxc-abc" {
		t.Errorf("Token = %q", c.Token)
	}
	if c.TeamName != "TestTeam" {
		t.Errorf("TeamName = %q", c.TeamName)
	}
}

func TestValidateTokens_MultipleTeams(t *testing.T) {
	teams := map[string][2]string{
		"xoxc-a":  {"T1", "Acme"},
		"xoxc-b":  {"T2", "Beta Corp"},
		"xoxc-a2": {"T1", "Acme"},
	}
	authTest := func(token, cookie string) (string, string, error) {
		if cookie != "xoxd-c" {
			t.Errorf("cookie = %q", cookie)
		}
		team, ok := teams[token]
		if !ok {
			return "", "", errors.New("invalid_auth")
		}
		return team[0], team[1], nil
	}

	creds, err := validateTokens([]string{"xoxc-a", "xoxc-stale", "xoxc-b", "xoxc-a2"}, "xoxd-c", authTest)
	if err != nil {
		t.Fatal(err)
	}
	if len(creds) != 2 {
		t.Fatalf("got %d creds, want 2", len(creds))
	}
	if creds[0].Token != "xoxc-a" || creds[0].TeamName != "Acme" {
		t.Errorf("creds[0] = %+v", creds[0])
	}
	if creds[1].Token != "xoxc-b" || creds[1].TeamID != "T2" {
		t.Errorf("creds[1] = %+v", creds[1])
	}
}

func TestValidateTokens_AllInvalid(t *testing.T) {
	authTest := func(token, cookie string) (string, string, error) {
		return "", "", errors.New("invalid_auth")
	}
	if _, err := validateTokens([]string{"xoxc-x"}, "", authTest); err == nil {
		t.Fatal("expected error")
	}
}
