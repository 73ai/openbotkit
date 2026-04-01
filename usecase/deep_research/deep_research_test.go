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

	prompt := "What is Qdrant and what makes it different from other vector databases? " +
		"Search for recent information and give me a brief summary."

	start := time.Now()
	result, err := a.Run(ctx, prompt)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	t.Logf("agent run took %s (response_len=%d)", time.Since(start).Round(time.Millisecond), len(result))

	spectest.AssertNotEmpty(t, result)
	spectest.AssertContains(t, result, "Qdrant")
	fx.AssertChecklist(t, result, []string{
		"Does the response explain what Qdrant is?",
		"Does it mention at least one distinguishing feature of Qdrant?",
		"Is the response a coherent summary, not raw search snippets?",
	})
}

func TestUseCase_DeepResearchDelegateTask(t *testing.T) {
	agents := tools.DetectAgents()
	if len(agents) == 0 {
		t.Skip("delegate_task tests require external AI CLI agent on PATH (claude, gemini, or codex)")
	}
	// Use only the highest-priority agent (claude > gemini > codex).
	agents = agents[:1]
	t.Logf("delegate agent: %s (%s)", agents[0].Kind, agents[0].Binary)

	fx := usecase.NewFixture(t)
	a := fx.AgentWithDelegation(t, agents)

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	prompt := "Compare the Raft and Paxos consensus algorithms. For each one, explain the core " +
		"mechanism, what failure scenarios it handles, and name a real system that uses it. " +
		"Then recommend which is better suited for a 5-node distributed key-value store."

	start := time.Now()
	result, err := a.Run(ctx, prompt)
	if err != nil {
		t.Fatalf("run: %v (elapsed %s)", err, time.Since(start).Round(time.Millisecond))
	}
	t.Logf("agent run took %s (response_len=%d)", time.Since(start).Round(time.Millisecond), len(result))

	spectest.AssertNotEmpty(t, result)
	spectest.AssertContains(t, result, "Raft", "Paxos")
	fx.AssertChecklist(t, result, []string{
		"Does the response explain the core mechanism of both Raft and Paxos?",
		"Does it discuss failure scenarios or fault tolerance?",
		"Does it name at least one real system that uses each algorithm?",
		"Does it make a recommendation for the key-value store scenario?",
	})
}
