package desktop

import "testing"

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
