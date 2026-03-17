package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/73ai/openbotkit/provider"
	"github.com/73ai/openbotkit/store"
)

const reconcilePromptTemplate = `Given existing memories and a new candidate fact, decide the action:
- ADD: new fact, no existing memory covers this
- UPDATE: existing memory covers same topic but needs updating (provide the ID to update and the new content)
- DELETE: new fact contradicts existing memory (provide the ID to delete)
- NOOP: fact already captured or same meaning

Existing memories:
%s

New fact: %s

Output JSON: {"action": "ADD|UPDATE|DELETE|NOOP", "id": 0, "content": ""}`

type reconcileDecision struct {
	Action  string `json:"action"`
	ID      int64  `json:"id"`
	Content string `json:"content"`
}

func Reconcile(ctx context.Context, db *store.DB, llm LLM, candidates []CandidateFact) (*ExtractResult, error) {
	result := &ExtractResult{}

	for _, candidate := range candidates {
		keywords := extractKeywords(candidate.Content)
		var existing []Memory
		for _, kw := range keywords {
			matches, err := Search(db, kw)
			if err != nil {
				continue
			}
			existing = dedup(existing, matches)
		}

		if len(existing) == 0 {
			_, err := Add(db, candidate.Content, Category(candidate.Category), "history", "")
			if err != nil {
				return result, fmt.Errorf("add memory: %w", err)
			}
			result.Added++
			continue
		}

		decision, err := reconcileWithLLM(ctx, llm, existing, candidate)
		if err != nil {
			result.Skipped++
			continue
		}

		switch strings.ToUpper(decision.Action) {
		case "ADD":
			_, err := Add(db, candidate.Content, Category(candidate.Category), "history", "")
			if err != nil {
				return result, fmt.Errorf("add memory: %w", err)
			}
			result.Added++
		case "UPDATE":
			content := decision.Content
			if content == "" {
				content = candidate.Content
			}
			if err := Update(db, decision.ID, content); err != nil {
				return result, fmt.Errorf("update memory: %w", err)
			}
			result.Updated++
		case "DELETE":
			if err := Delete(db, decision.ID); err != nil {
				return result, fmt.Errorf("delete memory: %w", err)
			}
			result.Deleted++
		default:
			result.Skipped++
		}
	}

	return result, nil
}

func reconcileWithLLM(ctx context.Context, llm LLM, existing []Memory, candidate CandidateFact) (*reconcileDecision, error) {
	var existingLines []string
	for _, m := range existing {
		existingLines = append(existingLines, fmt.Sprintf("[ID=%d] %s", m.ID, m.Content))
	}
	existingText := strings.Join(existingLines, "\n")

	prompt := fmt.Sprintf(reconcilePromptTemplate, existingText, candidate.Content)

	req := provider.ChatRequest{
		Messages: []provider.Message{
			provider.NewTextMessage(provider.RoleUser, prompt),
		},
		MaxTokens: 256,
	}

	resp, err := llm.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("reconciliation LLM call: %w", err)
	}

	return parseReconcileResponse(resp.TextContent())
}

func parseReconcileResponse(text string) (*reconcileDecision, error) {
	text = strings.TrimSpace(text)

	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start == -1 || end == -1 || end <= start {
		return &reconcileDecision{Action: "NOOP"}, nil
	}

	var d reconcileDecision
	if err := json.Unmarshal([]byte(text[start:end+1]), &d); err != nil {
		return &reconcileDecision{Action: "NOOP"}, nil
	}
	return &d, nil
}

func extractKeywords(content string) []string {
	words := strings.Fields(strings.ToLower(content))
	var keywords []string
	for _, w := range words {
		w = strings.Trim(w, ".,!?;:'\"")
		if len(w) > 3 && !isStopWord(w) {
			keywords = append(keywords, w)
		}
		if len(keywords) >= 3 {
			break
		}
	}
	return keywords
}

var stopWords = map[string]bool{
	"user": true, "that": true, "this": true, "with": true,
	"from": true, "they": true, "their": true, "have": true,
	"been": true, "will": true, "does": true, "about": true,
	"over": true, "also": true, "very": true, "just": true,
}

func isStopWord(w string) bool {
	return stopWords[w]
}

func dedup(existing, newItems []Memory) []Memory {
	seen := make(map[int64]bool)
	for _, m := range existing {
		seen[m.ID] = true
	}
	result := existing
	for _, m := range newItems {
		if !seen[m.ID] {
			result = append(result, m)
			seen[m.ID] = true
		}
	}
	return result
}
