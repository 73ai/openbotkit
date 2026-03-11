package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/priyanshujain/openbotkit/source/slack"
)

func TestSlackEditTool_Approved(t *testing.T) {
	api := &mockSlackAPI{channels: []slack.Channel{{ID: "C123", Name: "general"}}}
	inter := &mockInteractor{approveAll: true}
	tool := NewSlackEditTool(SlackToolDeps{Client: api, Interactor: inter})

	input, _ := json.Marshal(slackEditInput{Channel: "C123", TS: "111.222", Text: "updated"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if result != `{"ok":true}` {
		t.Errorf("result = %q", result)
	}
	if api.updatedTS != "111.222" {
		t.Errorf("updated ts = %q", api.updatedTS)
	}
}

func TestSlackEditTool_Denied(t *testing.T) {
	api := &mockSlackAPI{channels: []slack.Channel{{ID: "C123", Name: "general"}}}
	inter := &mockInteractor{approveAll: false}
	tool := NewSlackEditTool(SlackToolDeps{Client: api, Interactor: inter})

	input, _ := json.Marshal(slackEditInput{Channel: "C123", TS: "111.222", Text: "updated"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if result != "denied_by_user" {
		t.Errorf("result = %q", result)
	}
	if api.updatedTS != "" {
		t.Error("should not have updated when denied")
	}
}

func TestSlackEditTool_MissingParams(t *testing.T) {
	tool := NewSlackEditTool(SlackToolDeps{Client: &mockSlackAPI{}, Interactor: &mockInteractor{}})
	input, _ := json.Marshal(slackEditInput{Channel: "C123", TS: "111"})
	_, err := tool.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for missing text")
	}
}
