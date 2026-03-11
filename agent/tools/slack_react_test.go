package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/priyanshujain/openbotkit/source/slack"
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

func TestSlackReactTool_Denied(t *testing.T) {
	api := &mockSlackAPI{channels: []slack.Channel{{ID: "C123", Name: "general"}}}
	inter := &mockInteractor{approveAll: false}
	tool := NewSlackReactTool(SlackToolDeps{Client: api, Interactor: inter})

	input, _ := json.Marshal(slackReactInput{Channel: "C123", TS: "111", Emoji: "thumbsup"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if result != "denied_by_user" {
		t.Errorf("result = %q", result)
	}
	if api.reactedEmoji != "" {
		t.Error("should not have reacted when denied")
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
