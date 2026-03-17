package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/73ai/openbotkit/source/slack"
)

type mockSlackAPI struct {
	searchMessagesResult *slack.SearchResult
	searchFilesResult    *slack.FileSearchResult
	historyResult        []slack.Message
	repliesResult        []slack.Message
	channels             []slack.Channel
	users                []slack.User
	postedChannel        string
	postedText           string
	postedTS             string
	updatedTS            string
	deletedTS            string
	reactedEmoji         string
	reactAction          string
	err                  error
}

func (m *mockSlackAPI) SearchMessages(_ context.Context, _ string, _ slack.SearchOptions) (*slack.SearchResult, error) {
	return m.searchMessagesResult, m.err
}
func (m *mockSlackAPI) SearchFiles(_ context.Context, _ string, _ slack.SearchOptions) (*slack.FileSearchResult, error) {
	return m.searchFilesResult, m.err
}
func (m *mockSlackAPI) ConversationsHistory(_ context.Context, _ string, _ slack.HistoryOptions) ([]slack.Message, error) {
	return m.historyResult, m.err
}
func (m *mockSlackAPI) ConversationsReplies(_ context.Context, _ string, _ string, _ slack.HistoryOptions) ([]slack.Message, error) {
	return m.repliesResult, m.err
}
func (m *mockSlackAPI) ConversationsList(context.Context) ([]slack.Channel, error) {
	return m.channels, m.err
}
func (m *mockSlackAPI) UsersList(context.Context) ([]slack.User, error) { return m.users, m.err }
func (m *mockSlackAPI) UsersInfo(context.Context, string) (*slack.User, error) {
	return nil, m.err
}
func (m *mockSlackAPI) PostMessage(_ context.Context, channel, text, _ string) (string, error) {
	m.postedChannel = channel
	m.postedText = text
	return m.postedTS, m.err
}
func (m *mockSlackAPI) UpdateMessage(_ context.Context, _, ts, _ string) error {
	m.updatedTS = ts
	return m.err
}
func (m *mockSlackAPI) DeleteMessage(_ context.Context, _, ts string) error {
	m.deletedTS = ts
	return m.err
}
func (m *mockSlackAPI) AddReaction(_ context.Context, _, _, emoji string) error {
	m.reactedEmoji = emoji
	m.reactAction = "add"
	return m.err
}
func (m *mockSlackAPI) RemoveReaction(_ context.Context, _, _, emoji string) error {
	m.reactedEmoji = emoji
	m.reactAction = "remove"
	return m.err
}
func (m *mockSlackAPI) ResolveChannel(_ context.Context, ref string) (string, error) {
	return ref, nil
}
func (m *mockSlackAPI) ResolveUser(_ context.Context, ref string) (string, error) {
	return ref, nil
}

func TestSlackSearchTool_Messages(t *testing.T) {
	api := &mockSlackAPI{
		searchMessagesResult: &slack.SearchResult{
			Total:    1,
			Messages: []slack.Message{{TS: "123", Text: "found it"}},
		},
	}
	tool := NewSlackSearchTool(SlackToolDeps{Client: api})

	input, _ := json.Marshal(slackSearchInput{Query: "test", Limit: 5})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result, "found it") {
		t.Errorf("result = %q", result)
	}
}

func TestSlackSearchTool_Files(t *testing.T) {
	api := &mockSlackAPI{
		searchFilesResult: &slack.FileSearchResult{
			Total: 1,
			Files: []slack.File{{ID: "F1", Name: "doc.pdf"}},
		},
	}
	tool := NewSlackSearchTool(SlackToolDeps{Client: api})

	input, _ := json.Marshal(slackSearchInput{Query: "test", Type: "files"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result, "doc.pdf") {
		t.Errorf("result = %q", result)
	}
}

func TestSlackSearchTool_EmptyQuery(t *testing.T) {
	tool := NewSlackSearchTool(SlackToolDeps{Client: &mockSlackAPI{}})
	input, _ := json.Marshal(slackSearchInput{Query: ""})
	_, err := tool.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for empty query")
	}
}

func TestSlackSearchTool_Name(t *testing.T) {
	tool := NewSlackSearchTool(SlackToolDeps{})
	if tool.Name() != "slack_search" {
		t.Errorf("Name() = %q", tool.Name())
	}
	if tool.Description() == "" {
		t.Error("empty description")
	}
	if !json.Valid(tool.InputSchema()) {
		t.Error("invalid input schema")
	}
}

func TestSlackSearchTool_InvalidJSON(t *testing.T) {
	tool := NewSlackSearchTool(SlackToolDeps{Client: &mockSlackAPI{}})
	_, err := tool.Execute(context.Background(), json.RawMessage(`{invalid`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestSlackSearchTool_DefaultLimit(t *testing.T) {
	api := &mockSlackAPI{
		searchMessagesResult: &slack.SearchResult{Messages: []slack.Message{}},
	}
	tool := NewSlackSearchTool(SlackToolDeps{Client: api})
	input, _ := json.Marshal(slackSearchInput{Query: "test"})
	_, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
}
