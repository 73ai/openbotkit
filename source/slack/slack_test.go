package slack

import (
	"context"
	"testing"

	"github.com/73ai/openbotkit/config"
	"github.com/zalando/go-keyring"
)

func TestSlack_Name(t *testing.T) {
	s := New(Config{})
	if s.Name() != "slack" {
		t.Errorf("Name() = %q", s.Name())
	}
}

func TestSlack_Status_NilConfig(t *testing.T) {
	s := New(Config{})
	st, err := s.Status(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if st.Connected {
		t.Error("should not be connected with nil config")
	}
}

func TestSlack_Status_WithWorkspaces(t *testing.T) {
	s := New(Config{Slack: &config.SlackConfig{
		DefaultWorkspace: "test",
		Workspaces: map[string]config.SlackWorkspace{
			"test": {TeamID: "T1", TeamName: "Test"},
		},
	}})
	st, err := s.Status(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if st.Connected {
		t.Error("should not be connected without valid credentials")
	}
}

func TestSlack_Status_Connected(t *testing.T) {
	keyring.MockInit()

	SaveCredentials("myteam", "xoxp-test", "")

	s := New(Config{Slack: &config.SlackConfig{
		DefaultWorkspace: "myteam",
		Workspaces: map[string]config.SlackWorkspace{
			"myteam": {TeamID: "T1", TeamName: "MyTeam"},
		},
	}})
	st, err := s.Status(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !st.Connected {
		t.Error("should be connected with valid credentials")
	}
	if len(st.Accounts) != 1 || st.Accounts[0] != "myteam" {
		t.Errorf("accounts = %v", st.Accounts)
	}
	if st.ItemCount != 0 {
		t.Errorf("item count = %d", st.ItemCount)
	}
}
