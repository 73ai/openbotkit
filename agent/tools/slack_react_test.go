package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/73ai/openbotkit/source/slack"
)

func TestSlackReactTool_AddApproved(t *testing.T) {
	api := &mockSlackAPI{channels: []slack.Channel{{ID: "C123", Name: "general"}}}
	inter := &mockInteractor{approveAll: true}
	tool := NewSlackReactTool(SlackToolDeps{Client: api, Interactor: inter})

	input, _ := json.Marshal(slackReactInput{Channel: "C123", TS: "111", Emoji: "thumbsup"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if result != `{"ok":true}` {
		t.Errorf("result = %q", result)
	}
	if api.reactedEmoji != "thumbsup" {
		t.Errorf("emoji = %q", api.reactedEmoji)
	}
	if api.reactAction != "add" {
		t.Errorf("action = %q", api.reactAction)
	}
}

func TestSlackReactTool_RemoveApproved(t *testing.T) {
	api := &mockSlackAPI{channels: []slack.Channel{{ID: "C123", Name: "general"}}}
	inter := &mockInteractor{approveAll: true}
	tool := NewSlackReactTool(SlackToolDeps{Client: api, Interactor: inter})

	input, _ := json.Marshal(slackReactInput{Channel: "C123", TS: "111", Emoji: "thumbsup", Action: "remove"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if result != `{"ok":true}` {
		t.Errorf("result = %q", result)
	}
	if api.reactAction != "remove" {
		t.Errorf("action = %q", api.reactAction)
	}
}

func TestSlackReactTool_AutoApproves(t *testing.T) {
	api := &mockSlackAPI{channels: []slack.Channel{{ID: "C123", Name: "general"}}}
	inter := &mockInteractor{approveAll: false} // approveAll=false shouldn't matter for RiskLow
	tool := NewSlackReactTool(SlackToolDeps{Client: api, Interactor: inter})

	input, _ := json.Marshal(slackReactInput{Channel: "C123", TS: "111", Emoji: "thumbsup"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if result != `{"ok":true}` {
		t.Errorf("result = %q, want ok (auto-approved)", result)
	}
	if len(inter.approvals) != 0 {
		t.Error("RiskLow should not request approval")
	}
	if api.reactedEmoji != "thumbsup" {
		t.Error("reaction should have been auto-approved")
	}
}

func TestSlackReactTool_MissingParams(t *testing.T) {
	tool := NewSlackReactTool(SlackToolDeps{Client: &mockSlackAPI{}, Interactor: &mockInteractor{}})
	input, _ := json.Marshal(slackReactInput{Channel: "C123", TS: "111"})
	_, err := tool.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for missing emoji")
	}
}

func TestSlackReactTool_Metadata(t *testing.T) {
	tool := NewSlackReactTool(SlackToolDeps{Client: &mockSlackAPI{}, Interactor: &mockInteractor{}})
	if tool.Name() != "slack_react" {
		t.Errorf("Name() = %q", tool.Name())
	}
	if tool.Description() == "" {
		t.Error("empty description")
	}
	if !json.Valid(tool.InputSchema()) {
		t.Error("invalid schema")
	}
}

func TestSlackReactTool_InvalidJSON(t *testing.T) {
	tool := NewSlackReactTool(SlackToolDeps{Client: &mockSlackAPI{}, Interactor: &mockInteractor{}})
	_, err := tool.Execute(context.Background(), json.RawMessage(`{bad`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestSlackReactTool_InvalidAction(t *testing.T) {
	tool := NewSlackReactTool(SlackToolDeps{Client: &mockSlackAPI{}, Interactor: &mockInteractor{}})
	input, _ := json.Marshal(slackReactInput{Channel: "C123", TS: "111", Emoji: "thumbsup", Action: "delete"})
	_, err := tool.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for invalid action")
	}
}

func TestSlackReactTool_ResolveError(t *testing.T) {
	api := &mockSlackAPI{channels: []slack.Channel{}}
	inter := &mockInteractor{approveAll: true}
	tool := NewSlackReactTool(SlackToolDeps{Client: api, Interactor: inter})
	input, _ := json.Marshal(slackReactInput{Channel: "#nonexistent", TS: "111", Emoji: "thumbsup"})
	_, err := tool.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for unresolvable channel")
	}
}
