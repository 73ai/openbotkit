package anthropic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/priyanshujain/openbotkit/provider"
)

const (
	defaultBaseURL = "https://api.anthropic.com"
	apiVersion     = "2023-06-01"
)

// Anthropic implements the provider.Provider interface using the Anthropic Messages API.
type Anthropic struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// Ensure Anthropic implements Provider at compile time.
var _ provider.Provider = (*Anthropic)(nil)

func init() {
	provider.RegisterFactory("anthropic", func(apiKey, baseURL string) provider.Provider {
		var opts []Option
		if baseURL != "" {
			opts = append(opts, WithBaseURL(baseURL))
		}
		return New(apiKey, opts...)
	})
}

// Option configures the Anthropic provider.
type Option func(*Anthropic)

// WithBaseURL sets a custom API base URL (useful for testing).
func WithBaseURL(url string) Option {
	return func(a *Anthropic) { a.baseURL = url }
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(c *http.Client) Option {
	return func(a *Anthropic) { a.client = c }
}

// New creates a new Anthropic provider.
func New(apiKey string, opts ...Option) *Anthropic {
	a := &Anthropic{
		apiKey:  apiKey,
		baseURL: defaultBaseURL,
		client:  http.DefaultClient,
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// Chat sends a non-streaming chat request.
func (a *Anthropic) Chat(ctx context.Context, req provider.ChatRequest) (*provider.ChatResponse, error) {
	body := a.buildRequest(req, false)

	respBody, err := a.doRequest(ctx, body)
	if err != nil {
		return nil, err
	}
	defer respBody.Close()

	var apiResp apiResponse
	if err := json.NewDecoder(respBody).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if apiResp.Type == "error" {
		return nil, fmt.Errorf("anthropic API error: %s: %s", apiResp.Error.Type, apiResp.Error.Message)
	}

	return a.parseResponse(&apiResp), nil
}

// StreamChat sends a streaming chat request and returns a channel of events.
func (a *Anthropic) StreamChat(ctx context.Context, req provider.ChatRequest) (<-chan provider.StreamEvent, error) {
	body := a.buildRequest(req, true)

	respBody, err := a.doRequest(ctx, body)
	if err != nil {
		return nil, err
	}

	ch := make(chan provider.StreamEvent, 64)
	go a.parseSSE(respBody, ch)
	return ch, nil
}

func (a *Anthropic) buildRequest(req provider.ChatRequest, stream bool) map[string]any {
	body := map[string]any{
		"model":      req.Model,
		"max_tokens": req.MaxTokens,
	}
	if req.MaxTokens == 0 {
		body["max_tokens"] = 4096
	}
	if req.System != "" {
		body["system"] = req.System
	}
	if stream {
		body["stream"] = true
	}

	// Convert messages.
	var msgs []map[string]any
	for _, m := range req.Messages {
		msgs = append(msgs, convertMessage(m))
	}
	body["messages"] = msgs

	// Convert tools.
	if len(req.Tools) > 0 {
		var tools []map[string]any
		for _, t := range req.Tools {
			tools = append(tools, map[string]any{
				"name":         t.Name,
				"description":  t.Description,
				"input_schema": json.RawMessage(t.InputSchema),
			})
		}
		body["tools"] = tools
	}

	return body
}

func convertMessage(m provider.Message) map[string]any {
	msg := map[string]any{
		"role": string(m.Role),
	}

	var content []map[string]any
	for _, block := range m.Content {
		switch block.Type {
		case provider.ContentText:
			content = append(content, map[string]any{
				"type": "text",
				"text": block.Text,
			})
		case provider.ContentToolUse:
			if block.ToolCall != nil {
				content = append(content, map[string]any{
					"type":  "tool_use",
					"id":    block.ToolCall.ID,
					"name":  block.ToolCall.Name,
					"input": json.RawMessage(block.ToolCall.Input),
				})
			}
		case provider.ContentToolResult:
			if block.ToolResult != nil {
				result := map[string]any{
					"type":        "tool_result",
					"tool_use_id": block.ToolResult.ToolUseID,
					"content":     block.ToolResult.Content,
				}
				if block.ToolResult.IsError {
					result["is_error"] = true
				}
				content = append(content, result)
			}
		}
	}

	// Single text block → use string content for simplicity.
	if len(content) == 1 && content[0]["type"] == "text" {
		msg["content"] = content[0]["text"]
	} else {
		msg["content"] = content
	}

	return msg
}

func (a *Anthropic) doRequest(ctx context.Context, body map[string]any) (io.ReadCloser, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/v1/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", a.apiKey)
	httpReq.Header.Set("anthropic-version", apiVersion)

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		var apiErr apiResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err == nil && apiErr.Error.Message != "" {
			return nil, fmt.Errorf("anthropic API error (HTTP %d): %s: %s", resp.StatusCode, apiErr.Error.Type, apiErr.Error.Message)
		}
		return nil, fmt.Errorf("anthropic API error: HTTP %d", resp.StatusCode)
	}

	return resp.Body, nil
}

func (a *Anthropic) parseResponse(resp *apiResponse) *provider.ChatResponse {
	result := &provider.ChatResponse{
		Usage: provider.Usage{
			InputTokens:  resp.Usage.InputTokens,
			OutputTokens: resp.Usage.OutputTokens,
		},
	}

	switch resp.StopReason {
	case "end_turn":
		result.StopReason = provider.StopEndTurn
	case "tool_use":
		result.StopReason = provider.StopToolUse
	case "max_tokens":
		result.StopReason = provider.StopMaxTokens
	}

	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			result.Content = append(result.Content, provider.ContentBlock{
				Type: provider.ContentText,
				Text: block.Text,
			})
		case "tool_use":
			result.Content = append(result.Content, provider.ContentBlock{
				Type: provider.ContentToolUse,
				ToolCall: &provider.ToolCall{
					ID:    block.ID,
					Name:  block.Name,
					Input: block.Input,
				},
			})
		}
	}

	return result
}

func (a *Anthropic) parseSSE(body io.ReadCloser, ch chan<- provider.StreamEvent) {
	defer close(ch)
	defer body.Close()

	scanner := bufio.NewScanner(body)
	var currentToolCall *provider.ToolCall

	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var event sseEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		switch event.Type {
		case "content_block_start":
			if event.ContentBlock.Type == "tool_use" {
				currentToolCall = &provider.ToolCall{
					ID:   event.ContentBlock.ID,
					Name: event.ContentBlock.Name,
				}
				ch <- provider.StreamEvent{
					Type:     provider.EventToolCallStart,
					ToolCall: currentToolCall,
				}
			}

		case "content_block_delta":
			if event.Delta.Type == "text_delta" {
				ch <- provider.StreamEvent{
					Type: provider.EventTextDelta,
					Text: event.Delta.Text,
				}
			} else if event.Delta.Type == "input_json_delta" {
				ch <- provider.StreamEvent{
					Type:  provider.EventToolCallDelta,
					Delta: event.Delta.PartialJSON,
				}
			}

		case "content_block_stop":
			if currentToolCall != nil {
				ch <- provider.StreamEvent{
					Type: provider.EventToolCallEnd,
				}
				currentToolCall = nil
			}

		case "message_stop":
			ch <- provider.StreamEvent{
				Type: provider.EventDone,
			}
			return

		case "message_delta":
			// Contains stop_reason and usage — we emit Done.
			ch <- provider.StreamEvent{
				Type: provider.EventDone,
			}
			return

		case "error":
			ch <- provider.StreamEvent{
				Type:  provider.EventError,
				Error: fmt.Errorf("stream error: %s", event.Error.Message),
			}
			return
		}
	}
}

// API types for JSON marshaling.

type apiResponse struct {
	Type       string         `json:"type"`
	Content    []apiContent   `json:"content"`
	StopReason string         `json:"stop_reason"`
	Usage      apiUsage       `json:"usage"`
	Error      apiError       `json:"error"`
}

type apiContent struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

type apiUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type apiError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type sseEvent struct {
	Type         string      `json:"type"`
	ContentBlock apiContent  `json:"content_block,omitempty"`
	Delta        sseDelta    `json:"delta,omitempty"`
	Error        apiError    `json:"error,omitempty"`
}

type sseDelta struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	PartialJSON string `json:"partial_json,omitempty"`
}
