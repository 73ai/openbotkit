package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/73ai/openbotkit/provider"
)

const extractionPrompt = `You are a Personal Information Organizer. Extract personal facts about the user from this conversation.

Rules:
- Extract only facts that are likely true for months or years
- Each fact should be 1 sentence, starting with "User"
- Skip ephemeral facts, greetings, technical questions about code
- Categorize each fact as: identity, preference, relationship, or project

Output JSON array: [{"content": "User prefers dark mode", "category": "preference"}]
If no personal facts found, return: []`

type CandidateFact struct {
	Content  string `json:"content"`
	Category string `json:"category"`
}

type ExtractResult struct {
	Added   int
	Updated int
	Deleted int
	Skipped int
}

func Extract(ctx context.Context, llm LLM, messages []string) ([]CandidateFact, error) {
	filtered := preFilter(messages)
	if len(filtered) == 0 {
		return nil, nil
	}

	conversationText := strings.Join(filtered, "\n---\n")

	req := provider.ChatRequest{
		System: extractionPrompt,
		Messages: []provider.Message{
			provider.NewTextMessage(provider.RoleUser, conversationText),
		},
		MaxTokens: 2048,
	}

	resp, err := llm.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("extraction LLM call: %w", err)
	}

	return parseExtractionResponse(resp.TextContent())
}

func preFilter(messages []string) []string {
	var result []string
	for _, msg := range messages {
		if len(strings.Fields(msg)) < 5 {
			continue
		}
		lower := strings.ToLower(strings.TrimSpace(msg))
		if isAck(lower) {
			continue
		}
		result = append(result, msg)
	}
	return result
}

var ackPrefixes = []string{
	"ok", "thanks", "thank you", "yes", "no", "sure", "got it",
	"sounds good", "great", "perfect", "right", "yep", "nope",
}

func isAck(lower string) bool {
	for _, prefix := range ackPrefixes {
		if lower == prefix || strings.HasPrefix(lower, prefix+".") || strings.HasPrefix(lower, prefix+"!") || strings.HasPrefix(lower, prefix+",") {
			return true
		}
	}
	return false
}

func parseExtractionResponse(text string) ([]CandidateFact, error) {
	text = strings.TrimSpace(text)

	// Try to find JSON array in the response.
	start := strings.Index(text, "[")
	end := strings.LastIndex(text, "]")
	if start == -1 || end == -1 || end <= start {
		return nil, nil
	}
	jsonStr := text[start : end+1]

	var facts []CandidateFact
	if err := json.Unmarshal([]byte(jsonStr), &facts); err != nil {
		return nil, fmt.Errorf("parse extraction JSON: %w", err)
	}
	return facts, nil
}
