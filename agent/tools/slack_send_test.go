package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/priyanshujain/openbotkit/source/slack"
)

func TestSlackSendTool_Approved(t *testing.T) {
	api := &mockSlackAPI{
		postedTS: "999.999",
		channels: []slack.Channel{{ID: "C123", Name: "general"}},
	}
	inter := &mockInteractor{approveAll: true}
	tool := NewSlackSendTool(SlackToolDeps{Client: api, Interactor: inter})

	input, _ := json.Marshal(slackSendInput{Channel: "C123", Text: "hello"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result, "999.999") {
		t.Errorf("result = %q", result)
	}
	if api.postedText != "hello" {
		t.Errorf("posted text = %q", api.postedText)
	}
	if len(inter.approvals) != 1 {
		t.Errorf("expected 1 approval, got %d", len(inter.approvals))
	}
}

func TestSlackSendTool_Denied(t *testing.T) {
	api := &mockSlackAPI{channels: []slack.Channel{{ID: "C123", Name: "general"}}}
	inter := &mockInteractor{approveAll: false}
	tool := NewSlackSendTool(SlackToolDeps{Client: api, Interactor: inter})

	input, _ := json.Marshal(slackSendInput{Channel: "C123", Text: "hello"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if result != "denied_by_user" {
		t.Errorf("result = %q", result)
	}
	if api.postedText != "" {
		t.Error("message should not have been sent when denied")
	}
}

func TestSlackSendTool_MissingParams(t *testing.T) {
	tool := NewSlackSendTool(SlackToolDeps{Client: &mockSlackAPI{}, Interactor: &mockInteractor{}})
	input, _ := json.Marshal(slackSendInput{Channel: "C123"})
	_, err := tool.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for missing text")
	}
}

func TestSlackSendTool_Name(t *testing.T) {
	tool := NewSlackSendTool(SlackToolDeps{Client: &mockSlackAPI{}})
	if tool.Name() != "slack_send" {
		t.Errorf("Name() = %q", tool.Name())
	}
}
