package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/73ai/openbotkit/source/slack"
)

func TestSlackReadChannelTool_Execute(t *testing.T) {
	api := &mockSlackAPI{
		historyResult: []slack.Message{
			{TS: "111", Text: "hello world", User: "U1"},
			{TS: "222", Text: "how are you", User: "U2"},
		},
		channels: []slack.Channel{{ID: "C123", Name: "general"}},
	}
	tool := NewSlackReadChannelTool(SlackToolDeps{Client: api})

	input, _ := json.Marshal(slackReadChannelInput{Channel: "C123", Limit: 10})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result, "hello world") {
		t.Errorf("result = %q", result)
	}
}

func TestSlackReadChannelTool_ResolvesName(t *testing.T) {
	api := &mockSlackAPI{
		historyResult: []slack.Message{{TS: "111", Text: "test"}},
		channels:      []slack.Channel{{ID: "C123", Name: "general"}},
	}
	tool := NewSlackReadChannelTool(SlackToolDeps{Client: api})

	input, _ := json.Marshal(slackReadChannelInput{Channel: "#general"})
	_, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSlackReadChannelTool_EmptyChannel(t *testing.T) {
	tool := NewSlackReadChannelTool(SlackToolDeps{Client: &mockSlackAPI{}})
	input, _ := json.Marshal(slackReadChannelInput{Channel: ""})
	_, err := tool.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for empty channel")
	}
}

func TestSlackReadChannelTool_Name(t *testing.T) {
	tool := NewSlackReadChannelTool(SlackToolDeps{Client: &mockSlackAPI{}})
	if tool.Name() != "slack_read_channel" {
		t.Errorf("Name() = %q", tool.Name())
	}
}

func TestSlackReadChannelTool_Metadata(t *testing.T) {
	tool := NewSlackReadChannelTool(SlackToolDeps{Client: &mockSlackAPI{}})
	if tool.Name() != "slack_read_channel" {
		t.Errorf("Name() = %q", tool.Name())
	}
	if tool.Description() == "" {
		t.Error("empty description")
	}
	if !json.Valid(tool.InputSchema()) {
		t.Error("invalid schema")
	}
}

func TestSlackReadChannelTool_InvalidJSON(t *testing.T) {
	tool := NewSlackReadChannelTool(SlackToolDeps{Client: &mockSlackAPI{}})
	_, err := tool.Execute(context.Background(), json.RawMessage(`{bad`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestSlackReadChannelTool_ResolveError(t *testing.T) {
	api := &mockSlackAPI{channels: []slack.Channel{}}
	tool := NewSlackReadChannelTool(SlackToolDeps{Client: api})
	input, _ := json.Marshal(slackReadChannelInput{Channel: "#nonexistent"})
	_, err := tool.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for unresolvable channel")
	}
}
