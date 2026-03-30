package deep_research

import (
	"context"
	"testing"
	"time"

	"github.com/73ai/openbotkit/agent/tools"
	"github.com/73ai/openbotkit/spectest"
	"github.com/73ai/openbotkit/usecase"
)

func TestUseCase_DeepResearchSubagent(t *testing.T) {
	fx := usecase.NewFixture(t)
	a := fx.Agent(t)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	prompt := "I'm evaluating vector databases for a side project. Can you research the current " +
		"landscape and compare Pinecone, Weaviate, and Qdrant? Cover pricing, self-hosting " +
		"options, and what developers are saying about each one."
	result, err := a.Run(ctx, prompt)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	spectest.AssertNotEmpty(t, result)
	fx.AssertJudge(t, prompt, result,
		"Response contains a substantive comparison of at least Pinecone, Weaviate, and Qdrant. "+
			"Must mention concrete details about at least two of: pricing, self-hosting, developer sentiment. "+
			"Should read as a coherent analysis, not raw search snippets.")
}

func TestUseCase_DeepResearchDelegateTask(t *testing.T) {
	agents := tools.DetectAgents()
	if len(agents) == 0 {
		t.Skip("delegate_task tests require external AI CLI agent on PATH (claude, gemini, or codex)")
	}

	fx := usecase.NewFixture(t)
	a := fx.AgentWithDelegation(t, agents)

	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
	defer cancel()

	prompt := "Research the current state of WebAssembly for server-side use cases. " +
		"Which runtimes are popular (Wasmtime, Wasmer, WasmEdge), what languages compile " +
		"to WASM well, and are there production use cases? Write a summary."
	result, err := a.Run(ctx, prompt)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	spectest.AssertNotEmpty(t, result)
	fx.AssertJudge(t, prompt, result,
		"Response should summarize WebAssembly server-side developments mentioning at least "+
			"two runtimes and at least one language that compiles to WASM.")
}
