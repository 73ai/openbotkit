package spectest

import (
	"context"
	"testing"
	"time"

	"github.com/priyanshujain/openbotkit/agent"
	"github.com/priyanshujain/openbotkit/agent/tools"
	"github.com/priyanshujain/openbotkit/source/websearch"
)

func webAgent(t *testing.T, fx *LocalFixture) *agent.Agent {
	t.Helper()

	ws := websearch.New(websearch.Config{})
	deps := tools.WebToolDeps{
		WS:       ws,
		Provider: fx.Provider,
		Model:    fx.Model,
	}

	toolReg := tools.NewRegistry()
	toolReg.Register(tools.NewWebSearchTool(deps))
	toolReg.Register(tools.NewWebFetchTool(deps))

	identity := "You are a personal AI assistant with web access.\n"
	blocks := tools.BuildSystemBlocks(identity, toolReg)

	return agent.New(fx.Provider, fx.Model, toolReg,
		agent.WithSystemBlocks(blocks),
		agent.WithMaxIterations(10),
	)
}

func TestSpec_WebSearch(t *testing.T) {
	EachProvider(t, func(t *testing.T, fx *LocalFixture) {
		a := webAgent(t, fx)
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		prompt := "Search for golang generics tutorial"
		result, err := a.Run(ctx, prompt)
		if err != nil {
			t.Fatalf("agent.Run: %v", err)
		}

		AssertNotEmpty(t, result)
		fx.AssertJudge(t, prompt, result,
			"Response must be about Go generics and contain useful information "+
				"such as type parameters, constraints, or tutorial content")
	})
}

func TestSpec_WebFetch(t *testing.T) {
	EachProvider(t, func(t *testing.T, fx *LocalFixture) {
		a := webAgent(t, fx)
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		prompt := "Fetch https://go.dev and tell me what Go is"
		result, err := a.Run(ctx, prompt)
		if err != nil {
			t.Fatalf("agent.Run: %v", err)
		}

		AssertNotEmpty(t, result)
		fx.AssertJudge(t, prompt, result,
			"Response must describe Go as a programming language based on content from go.dev. "+
				"It should NOT dump raw HTML or markdown.")
	})
}

func TestSpec_WebSearchThenFetch(t *testing.T) {
	EachProvider(t, func(t *testing.T, fx *LocalFixture) {
		a := webAgent(t, fx)
		ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
		defer cancel()

		prompt := "Search for the Go programming language homepage, then fetch the top result and summarize it"
		result, err := a.Run(ctx, prompt)
		if err != nil {
			t.Fatalf("agent.Run: %v", err)
		}

		AssertNotEmpty(t, result)
		fx.AssertJudge(t, prompt, result,
			"Response must provide a summary about the Go programming language "+
				"that includes concrete details such as features, use cases, or design goals")
	})
}
