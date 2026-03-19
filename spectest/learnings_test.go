package spectest

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/73ai/openbotkit/agent"
	"github.com/73ai/openbotkit/agent/tools"
	"github.com/73ai/openbotkit/service/learnings"
)

func agentWithLearningsTools(t *testing.T, fx *LocalFixture) (*agent.Agent, string) {
	t.Helper()

	dir := filepath.Join(fx.dir, "learnings")

	st := learnings.New(dir)
	deps := tools.LearningsDeps{
		Store: st,
	}

	toolReg := tools.NewRegistry()
	toolReg.Register(tools.NewBashTool(30 * time.Second))
	toolReg.Register(&tools.FileReadTool{})
	toolReg.Register(&tools.LoadSkillsTool{})
	toolReg.Register(&tools.SearchSkillsTool{})
	toolReg.Register(tools.NewLearningSaveTool(deps))
	toolReg.Register(tools.NewLearningReadTool(deps))
	toolReg.Register(tools.NewLearningSearchTool(deps))

	identity := "You are a personal AI assistant powered by OpenBotKit.\n"
	blocks := tools.BuildSystemBlocks(identity, toolReg)

	a := agent.New(fx.Provider, fx.Model, toolReg,
		agent.WithSystemBlocks(blocks),
		agent.WithMaxIterations(10),
	)
	return a, dir
}

func TestSpec_LearningSaveAndRead(t *testing.T) {
	EachProvider(t, func(t *testing.T, fx *LocalFixture) {
		a, dir := agentWithLearningsTools(t, fx)

		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		prompt := "Save this learning about Go: goroutines are multiplexed onto OS threads by the Go scheduler, and a buffered channel with capacity 1 can act like a mutex"
		result, err := a.Run(ctx, prompt)
		if err != nil {
			t.Fatalf("agent.Run: %v", err)
		}
		AssertNotEmpty(t, result)
		fx.AssertJudge(t, prompt, result,
			"The response should confirm that a learning about Go was saved.")

		files, _ := filepath.Glob(filepath.Join(dir, "*.md"))
		if len(files) == 0 {
			t.Fatal("expected at least one .md file in learnings dir")
		}

		readPrompt := "What are my saved learnings about Go?"
		readResult, err := a.Run(ctx, readPrompt)
		if err != nil {
			t.Fatalf("agent.Run (read): %v", err)
		}
		AssertNotEmpty(t, readResult)
		fx.AssertJudge(t, readPrompt, readResult,
			"The response should mention goroutines and channels or mutex, reflecting the previously saved learning.")
	})
}

func TestSpec_LearningSaveAndSearch(t *testing.T) {
	EachProvider(t, func(t *testing.T, fx *LocalFixture) {
		a, _ := agentWithLearningsTools(t, fx)

		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		_, err := a.Run(ctx, "Save a learning about SQL: composite indexes should have the most selective column first")
		if err != nil {
			t.Fatalf("save SQL: %v", err)
		}
		_, err = a.Run(ctx, "Save a learning about Docker: multi-stage builds reduce final image size by discarding build dependencies")
		if err != nil {
			t.Fatalf("save Docker: %v", err)
		}

		searchPrompt := "Search my learnings for anything about indexes"
		result, err := a.Run(ctx, searchPrompt)
		if err != nil {
			t.Fatalf("search: %v", err)
		}
		AssertNotEmpty(t, result)
		fx.AssertJudge(t, searchPrompt, result,
			"The response should mention SQL or composite indexes. It should not mention Docker.")
	})
}

func TestSpec_LearningListTopics(t *testing.T) {
	EachProvider(t, func(t *testing.T, fx *LocalFixture) {
		a, _ := agentWithLearningsTools(t, fx)

		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		_, _ = a.Run(ctx, "Save a learning about Python: list comprehensions are faster than for loops for simple transformations")
		_, _ = a.Run(ctx, "Save a learning about Kubernetes: pods are the smallest deployable unit")

		listPrompt := "What learning topics do I have saved?"
		result, err := a.Run(ctx, listPrompt)
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		AssertNotEmpty(t, result)
		fx.AssertJudge(t, listPrompt, result,
			"The response should list at least two topics, including Python and Kubernetes.")
	})
}
