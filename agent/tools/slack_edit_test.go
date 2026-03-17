package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/73ai/openbotkit/source/slack"
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

func TestSlackEditTool_Metadata(t *testing.T) {
	tool := NewSlackEditTool(SlackToolDeps{Client: &mockSlackAPI{}, Interactor: &mockInteractor{}})
	if tool.Name() != "slack_edit" {
		t.Errorf("Name() = %q", tool.Name())
	}
	if tool.Description() == "" {
		t.Error("empty description")
	}
	if !json.Valid(tool.InputSchema()) {
		t.Error("invalid schema")
	}
}

func TestSlackEditTool_InvalidJSON(t *testing.T) {
	tool := NewSlackEditTool(SlackToolDeps{Client: &mockSlackAPI{}, Interactor: &mockInteractor{}})
	_, err := tool.Execute(context.Background(), json.RawMessage(`{bad`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestSlackEditTool_ResolveError(t *testing.T) {
	api := &mockSlackAPI{channels: []slack.Channel{}}
	inter := &mockInteractor{approveAll: true}
	tool := NewSlackEditTool(SlackToolDeps{Client: api, Interactor: inter})
	input, _ := json.Marshal(slackEditInput{Channel: "#nonexistent", TS: "111", Text: "updated"})
	_, err := tool.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for unresolvable channel")
	}
}
