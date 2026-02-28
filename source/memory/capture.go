package memory

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/priyanshujain/openbotkit/store"
)

// transcriptLine represents a single line from a Claude Code JSONL transcript.
type transcriptLine struct {
	Type    string          `json:"type"`
	Message json.RawMessage `json:"message"`
}

type transcriptMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func Capture(db *store.DB, input CaptureInput) error {
	if err := Migrate(db); err != nil {
		return fmt.Errorf("migrate schema: %w", err)
	}

	convID, err := UpsertConversation(db, input.SessionID, input.CWD)
	if err != nil {
		return fmt.Errorf("upsert conversation: %w", err)
	}

	existingCount, err := MessageCountForSession(db, input.SessionID)
	if err != nil {
		return fmt.Errorf("count existing messages: %w", err)
	}

	messages, err := parseTranscript(input.TranscriptPath)
	if err != nil {
		return fmt.Errorf("parse transcript: %w", err)
	}

	// Skip already-stored messages for idempotency.
	if existingCount >= len(messages) {
		return nil
	}
	newMessages := messages[existingCount:]

	for _, msg := range newMessages {
		if err := SaveMessage(db, convID, msg.role, msg.content); err != nil {
			return fmt.Errorf("save message: %w", err)
		}
	}

	return nil
}

type parsedMessage struct {
	role    string
	content string
}

func parseTranscript(path string) ([]parsedMessage, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open transcript: %w", err)
	}
	defer f.Close()

	var messages []parsedMessage
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 100*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		var tl transcriptLine
		if err := json.Unmarshal(line, &tl); err != nil {
			continue
		}

		if tl.Type != "user" && tl.Type != "assistant" {
			continue
		}
		if tl.Message == nil {
			continue
		}

		var msg transcriptMessage
		if err := json.Unmarshal(tl.Message, &msg); err != nil {
			continue
		}

		text := extractText(msg.Content)
		if text == "" {
			continue
		}

		role := tl.Type
		if role == "user" {
			// Skip tool results and system messages.
			if isToolResult(msg.Content) || isSystemMessage(text) {
				continue
			}
		}
		if role == "assistant" {
			// Skip pure tool-use-only turns (no text).
			if text == "" {
				continue
			}
		}

		messages = append(messages, parsedMessage{role: role, content: text})
	}

	return messages, scanner.Err()
}

func extractText(raw json.RawMessage) string {
	// Try as plain string first.
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.TrimSpace(s)
	}

	// Try as array of content blocks.
	var blocks []contentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return ""
	}

	var texts []string
	for _, b := range blocks {
		if b.Type == "text" && b.Text != "" {
			texts = append(texts, b.Text)
		}
	}
	return strings.TrimSpace(strings.Join(texts, "\n"))
}

func isToolResult(raw json.RawMessage) bool {
	var blocks []contentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return false
	}
	for _, b := range blocks {
		if b.Type == "tool_result" {
			return true
		}
	}
	return false
}

func isSystemMessage(text string) bool {
	return strings.HasPrefix(text, "<local-command") ||
		strings.HasPrefix(text, "<command-name>") ||
		strings.HasPrefix(text, "<system-reminder>")
}
