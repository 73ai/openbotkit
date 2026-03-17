package spectest

import (
	"context"
	"testing"
	"time"

	"github.com/73ai/openbotkit/agent"
	"github.com/73ai/openbotkit/agent/tools"
	"github.com/73ai/openbotkit/source/slack"
	"github.com/73ai/openbotkit/source/slack/desktop"
)

// slackClient extracts credentials from the running Slack Desktop app.
// Returns nil if Slack Desktop is not installed or credentials are invalid.
func slackClient(t *testing.T) *slack.Client {
	t.Helper()
	creds, err := desktop.Extract()
	if err != nil {
		t.Skipf("slack desktop not available: %v", err)
	}
	return slack.NewClient(creds.Token, creds.Cookie)
}

// slackAgent creates an agent with only read-only Slack tools registered.
// No write tools (send/edit/react) are included to keep tests safe.
func slackAgent(t *testing.T, fx *LocalFixture, client slack.API) *agent.Agent {
	t.Helper()

	deps := tools.SlackToolDeps{Client: client}
	toolReg := tools.NewRegistry()
	toolReg.Register(tools.NewSlackSearchTool(deps))
	toolReg.Register(tools.NewSlackReadChannelTool(deps))
	toolReg.Register(tools.NewSlackReadThreadTool(deps))

	identity := "You are a personal AI assistant with access to Slack.\n"
	blocks := tools.BuildSystemBlocks(identity, toolReg)

	return agent.New(fx.Provider, fx.Model, toolReg,
		agent.WithSystemBlocks(blocks),
		agent.WithMaxIterations(10),
	)
}

func TestSpec_SlackListChannels(t *testing.T) {
	client := slackClient(t)

	EachProvider(t, func(t *testing.T, fx *LocalFixture) {
		a := slackAgent(t, fx, client)
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		prompt := "List the Slack channels available in this workspace. Just give me the channel names."
		result, err := a.Run(ctx, prompt)
		if err != nil {
			t.Fatalf("agent.Run: %v", err)
		}

		AssertNotEmpty(t, result)
		fx.AssertJudge(t, prompt, result,
			"The response must list one or more Slack channel names. "+
				"It should include real channel names from the workspace (e.g. #general or similar).")
	})
}

func TestSpec_SlackSearchMessages(t *testing.T) {
	client := slackClient(t)

	EachProvider(t, func(t *testing.T, fx *LocalFixture) {
		a := slackAgent(t, fx, client)
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		prompt := "Search Slack for recent messages. Show me a few results with who said what."
		result, err := a.Run(ctx, prompt)
		if err != nil {
			t.Fatalf("agent.Run: %v", err)
		}

		AssertNotEmpty(t, result)
		fx.AssertJudge(t, prompt, result,
			"The response must include at least one Slack message with some indication "+
				"of the sender or content. It should not say it cannot access Slack.")
	})
}
