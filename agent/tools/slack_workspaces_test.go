package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/source/slack"
	"github.com/zalando/go-keyring"
)

func twoWorkspaceDeps(def, other *mockSlackAPI, inter Interactor) SlackToolDeps {
	return SlackToolDeps{
		Client:     def,
		Workspace:  "acme",
		Interactor: inter,
		Workspaces: []*SlackWorkspace{NewSlackWorkspace("beta-corp", "Beta Corp", other)},
	}
}

func TestSlackSend_RoutesToNamedWorkspace(t *testing.T) {
	def := &mockSlackAPI{postedTS: "1.1"}
	other := &mockSlackAPI{postedTS: "2.2"}
	inter := &mockInteractor{approveAll: true}
	tool := NewSlackSendTool(twoWorkspaceDeps(def, other, inter))

	input, _ := json.Marshal(slackSendInput{Workspace: "Beta Corp", Channel: "C123", Text: "hi"})
	if _, err := tool.Execute(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	if other.postedChannel != "C123" || def.postedChannel != "" {
		t.Errorf("posted default=%q other=%q", def.postedChannel, other.postedChannel)
	}
	if len(inter.approvals) != 1 || !strings.Contains(inter.approvals[0], "beta-corp") {
		t.Errorf("approval prompt should name the workspace: %v", inter.approvals)
	}
}

func TestSlackSend_DefaultsToDefaultWorkspace(t *testing.T) {
	def := &mockSlackAPI{postedTS: "1.1"}
	other := &mockSlackAPI{}
	tool := NewSlackSendTool(twoWorkspaceDeps(def, other, &mockInteractor{approveAll: true}))

	input, _ := json.Marshal(slackSendInput{Channel: "C123", Text: "hi"})
	if _, err := tool.Execute(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	if def.postedChannel != "C123" || other.postedChannel != "" {
		t.Errorf("posted default=%q other=%q", def.postedChannel, other.postedChannel)
	}
}

func TestSlackSearch_UnknownWorkspace(t *testing.T) {
	tool := NewSlackSearchTool(twoWorkspaceDeps(&mockSlackAPI{}, &mockSlackAPI{}, nil))

	input, _ := json.Marshal(slackSearchInput{Workspace: "nope", Query: "x"})
	_, err := tool.Execute(context.Background(), input)
	if err == nil || !strings.Contains(err.Error(), "acme, beta-corp") {
		t.Fatalf("err = %v", err)
	}
}

func TestSlackReadThread_PermalinkRoutesBySubdomain(t *testing.T) {
	thread := []slack.Message{{TS: "1785334150.236249", Text: "parent"}}
	def := &mockSlackAPI{}
	other := &mockSlackAPI{repliesResult: thread}
	tool := NewSlackReadThreadTool(twoWorkspaceDeps(def, other, nil))

	input, _ := json.Marshal(slackReadThreadInput{
		Permalink: "https://betacorp.slack.com/archives/C01Q9G9CD6X/p1785334150236249",
	})
	if _, err := tool.Execute(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	if len(other.repliesTS) != 1 || len(def.repliesTS) != 0 {
		t.Errorf("replies default=%v other=%v", def.repliesTS, other.repliesTS)
	}
}

func TestSlackTools_DescribeWorkspaces(t *testing.T) {
	multi := NewSlackSearchTool(twoWorkspaceDeps(&mockSlackAPI{}, &mockSlackAPI{}, nil))
	if !strings.Contains(multi.Description(), "acme, beta-corp (default: acme)") {
		t.Errorf("description = %q", multi.Description())
	}
	single := NewSlackSearchTool(SlackToolDeps{Client: &mockSlackAPI{}, Workspace: "acme"})
	if strings.Contains(single.Description(), "workspaces") {
		t.Errorf("single-workspace description = %q", single.Description())
	}
}

func TestSlackTools_SchemasAreValidJSON(t *testing.T) {
	deps := SlackToolDeps{Client: &mockSlackAPI{}}
	for _, tool := range []Tool{
		NewSlackSearchTool(deps), NewSlackReadChannelTool(deps), NewSlackReadThreadTool(deps),
		NewSlackMediaDownloadTool(deps), NewSlackSendTool(deps), NewSlackEditTool(deps), NewSlackReactTool(deps),
	} {
		var v map[string]any
		if err := json.Unmarshal(tool.InputSchema(), &v); err != nil {
			t.Errorf("%s: %v", tool.Name(), err)
		}
	}
}

func TestSlackReadThread_PermalinkFromUnconnectedWorkspace(t *testing.T) {
	tool := NewSlackReadThreadTool(twoWorkspaceDeps(&mockSlackAPI{}, &mockSlackAPI{}, nil))

	input, _ := json.Marshal(slackReadThreadInput{
		Permalink: "https://other.slack.com/archives/C01Q9G9CD6X/p1785334150236249",
	})
	_, err := tool.Execute(context.Background(), input)
	if err == nil || !strings.Contains(err.Error(), `"other"`) {
		t.Fatalf("err = %v", err)
	}
}

func TestLoadSlackToolDeps(t *testing.T) {
	keyring.MockInit()
	cfg := &config.SlackConfig{
		DefaultWorkspace: "beta-corp",
		Workspaces: map[string]config.SlackWorkspace{
			"acme":      {TeamName: "Acme"},
			"beta-corp": {TeamName: "Beta Corp"},
			"gone":      {TeamName: "Gone"},
		},
	}
	if _, ok := LoadSlackToolDeps(cfg); ok {
		t.Fatal("expected ok=false without default workspace credentials")
	}
	slack.SaveCredentials("acme", "xoxc-a", "")
	slack.SaveCredentials("beta-corp", "xoxc-b", "")

	deps, ok := LoadSlackToolDeps(cfg)
	if !ok {
		t.Fatal("expected deps")
	}
	r := deps.router()
	if r.names() != "beta-corp, acme" || r[0].TeamName != "Beta Corp" {
		t.Errorf("workspaces = %s", r.names())
	}
}
