package spectest

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/73ai/openbotkit/provider"
)

// AssertJudge uses the fixture's dedicated judge provider to evaluate whether
// the agent's response satisfies the given criteria. Using a separate judge
// avoids self-evaluation bias (e.g., Gemini misjudging its own correct output).
func (f *LocalFixture) AssertJudge(t *testing.T, prompt, response, criteria string) {
	t.Helper()

	judgePrompt := `You are a strict test evaluator. You will be given:
1. The user's original question
2. The AI assistant's response
3. Success criteria

The assistant had access to various tools (databases, web search, web fetch, etc.) via tool calls.
The details in the response come from those tools — they are NOT hallucinated.
Your job is ONLY to check whether the response covers the topics described in the criteria.

Respond with exactly one line: "PASS" or "FAIL"
Then on the next line, a brief explanation (1-2 sentences).

User question: ` + prompt + `

Assistant response:
` + response + `

Success criteria: ` + criteria

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := f.JudgeProvider.Chat(ctx, provider.ChatRequest{
		Model: f.JudgeModel,
		Messages: []provider.Message{
			provider.NewTextMessage(provider.RoleUser, judgePrompt),
		},
		MaxTokens: 1024,
	})
	if err != nil {
		t.Fatalf("judge LLM call failed: %v", err)
	}

	verdict := resp.TextContent()
	firstLine := strings.SplitN(strings.TrimSpace(verdict), "\n", 2)[0]
	firstLine = strings.TrimSpace(firstLine)

	if !strings.EqualFold(firstLine, "PASS") {
		t.Errorf("judge FAIL for criteria %q\njudge said: %s\nagent response was:\n%s", criteria, verdict, response)
	}
}

// AssertChecklist sends binary yes/no questions to the judge LLM and fails
// the test for any question answered NO. This is more reliable than open-ended
// criteria for complex, multi-dimension evaluation (CheckEval pattern).
func (f *LocalFixture) AssertChecklist(t *testing.T, response string, questions []string) {
	t.Helper()

	if len(questions) == 0 {
		t.Fatal("AssertChecklist: no questions provided")
	}

	// Truncate response to keep judge prompt fast and within budget.
	const maxChars = 4000
	truncated := response
	if len(truncated) > maxChars {
		truncated = truncated[:maxChars]
	}

	var numbered strings.Builder
	for i, q := range questions {
		fmt.Fprintf(&numbered, "%d. %s\n", i+1, q)
	}

	judgePrompt := `You are evaluating an AI assistant's response. Answer each question with YES or NO only.

Response to evaluate:
` + truncated + `

Questions:
` + numbered.String() + `
Answer each with the question number followed by YES or NO. Example:
1. YES
2. NO`

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := f.JudgeProvider.Chat(ctx, provider.ChatRequest{
		Model: f.JudgeModel,
		Messages: []provider.Message{
			provider.NewTextMessage(provider.RoleUser, judgePrompt),
		},
		MaxTokens: 256,
	})
	if err != nil {
		t.Fatalf("checklist judge LLM call failed: %v", err)
	}

	verdict := resp.TextContent()
	var failed []string
	for i, q := range questions {
		prefix := fmt.Sprintf("%d.", i+1)
		found := false
		for _, line := range strings.Split(verdict, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, prefix) {
				found = true
				answer := strings.TrimSpace(strings.TrimPrefix(line, prefix))
				if !strings.HasPrefix(strings.ToUpper(answer), "YES") {
					failed = append(failed, fmt.Sprintf("  %d. %s → %s", i+1, q, answer))
				}
				break
			}
		}
		if !found {
			failed = append(failed, fmt.Sprintf("  %d. %s → (no answer from judge)", i+1, q))
		}
	}

	if len(failed) > 0 {
		t.Errorf("checklist judge failed:\n%s\njudge raw output:\n%s",
			strings.Join(failed, "\n"), verdict)
	}
}
