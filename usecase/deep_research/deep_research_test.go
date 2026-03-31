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

	start := time.Now()
	result, err := a.Run(ctx, prompt)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	t.Logf("agent run took %s (response_len=%d)", time.Since(start).Round(time.Millisecond), len(result))

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
	// Use only the highest-priority agent (claude > gemini > codex).
	// Passing all agents lets the LLM pick freely; gemini CLI is too slow
	// for CI (5-9 min vs ~45s for the same research task).
	agents = agents[:1]
	t.Logf("delegate agent: %s (%s)", agents[0].Kind, agents[0].Binary)

	fx := usecase.NewFixture(t)
	a := fx.AgentWithDelegation(t, agents)

	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
	defer cancel()

	prompt := "Research the current state of WebAssembly for server-side use cases. " +
		"Which runtimes are popular (Wasmtime, Wasmer, WasmEdge), what languages compile " +
		"to WASM well, and are there production use cases? Give me the full summary directly."

	start := time.Now()
	result, err := a.Run(ctx, prompt)
	if err != nil {
		t.Fatalf("run: %v (elapsed %s)", err, time.Since(start).Round(time.Millisecond))
	}
	t.Logf("agent run took %s (response_len=%d)", time.Since(start).Round(time.Millisecond), len(result))

	spectest.AssertNotEmpty(t, result)
	spectest.AssertContains(t, result, "Wasmtime", "Wasmer")
	fx.AssertChecklist(t, result, []string{
		"Does the response mention at least two WebAssembly runtimes?",
		"Does it mention at least one language that compiles to WASM?",
		"Does it discuss server-side or backend use cases for WebAssembly?",
	})
}
