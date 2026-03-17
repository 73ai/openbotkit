package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/73ai/openbotkit/source/slack"
)

func TestSlackReadThreadTool_Execute(t *testing.T) {
	api := &mockSlackAPI{
		repliesResult: []slack.Message{
			{TS: "111", Text: "parent message"},
			{TS: "222", Text: "reply 1", ThreadTS: "111"},
		},
		channels: []slack.Channel{{ID: "C123", Name: "general"}},
	}
	tool := NewSlackReadThreadTool(SlackToolDeps{Client: api})

	input, _ := json.Marshal(slackReadThreadInput{Channel: "C123", ThreadTS: "111"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result, "reply 1") {
		t.Errorf("result = %q", result)
	}
}

func TestSlackReadThreadTool_MissingParams(t *testing.T) {
	tool := NewSlackReadThreadTool(SlackToolDeps{Client: &mockSlackAPI{}})

	input, _ := json.Marshal(slackReadThreadInput{Channel: "C123"})
	_, err := tool.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for missing thread_ts")
	}
}

func TestSlackReadThreadTool_Name(t *testing.T) {
	tool := NewSlackReadThreadTool(SlackToolDeps{Client: &mockSlackAPI{}})
	if tool.Name() != "slack_read_thread" {
		t.Errorf("Name() = %q", tool.Name())
	}
}

func TestSlackReadThreadTool_Metadata(t *testing.T) {
	tool := NewSlackReadThreadTool(SlackToolDeps{Client: &mockSlackAPI{}})
	if tool.Name() != "slack_read_thread" {
		t.Errorf("Name() = %q", tool.Name())
	}
	if tool.Description() == "" {
		t.Error("empty description")
	}
	if !json.Valid(tool.InputSchema()) {
		t.Error("invalid schema")
	}
}

func TestSlackReadThreadTool_InvalidJSON(t *testing.T) {
	tool := NewSlackReadThreadTool(SlackToolDeps{Client: &mockSlackAPI{}})
	_, err := tool.Execute(context.Background(), json.RawMessage(`{bad`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestSlackReadThreadTool_ResolveError(t *testing.T) {
	api := &mockSlackAPI{channels: []slack.Channel{}}
	tool := NewSlackReadThreadTool(SlackToolDeps{Client: api})
	input, _ := json.Marshal(slackReadThreadInput{Channel: "#nonexistent", ThreadTS: "111"})
	_, err := tool.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for unresolvable channel")
	}
}
