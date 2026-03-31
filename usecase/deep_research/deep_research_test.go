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
	spectest.AssertContains(t, result, "Pinecone", "Weaviate", "Qdrant")
	fx.AssertChecklist(t, result, []string{
		"Does the response compare at least three vector databases?",
		"Does it discuss pricing or cost for at least one database?",
		"Does it discuss self-hosting options for at least one database?",
		"Is the response written as coherent analysis, not raw search result snippets?",
	})
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
		"to WASM well, and are there production use cases? Give me the full summary directly."
	result, err := a.Run(ctx, prompt)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	spectest.AssertNotEmpty(t, result)
	spectest.AssertContains(t, result, "Wasmtime", "Wasmer")
	fx.AssertChecklist(t, result, []string{
		"Does the response mention at least two WebAssembly runtimes?",
		"Does it mention at least one language that compiles to WASM?",
		"Does it discuss server-side or backend use cases for WebAssembly?",
	})
}
