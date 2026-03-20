package zai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/provider"
)

const defaultBaseURL = "https://api.z.ai/api/paas/v4"

// ZAI implements provider.Provider using the Z.AI GLM API.
type ZAI struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

var _ provider.Provider = (*ZAI)(nil)

type Option func(*ZAI)

func WithBaseURL(url string) Option {
	return func(z *ZAI) { z.baseURL = url }
}

func WithHTTPClient(c *http.Client) Option {
	return func(z *ZAI) { z.client = c }
}

func New(apiKey string, opts ...Option) *ZAI {
	z := &ZAI{
		apiKey:  apiKey,
		baseURL: defaultBaseURL,
		client:  &http.Client{Timeout: 60 * time.Second},
	}
	for _, opt := range opts {
		opt(z)
	}
	return z
}

func init() {
	provider.RegisterFactory("zai", func(cfg config.ModelProviderConfig, apiKey string) provider.Provider {
		var opts []Option
		if cfg.BaseURL != "" {
			opts = append(opts, WithBaseURL(cfg.BaseURL))
		}
		return New(apiKey, opts...)
	})
}

func (z *ZAI) Chat(ctx context.Context, req provider.ChatRequest) (*provider.ChatResponse, error) {
	body := z.buildRequest(req, false)

	respBody, err := z.doRequest(ctx, body)
	if err != nil {
		return nil, err
	}
	defer respBody.Close()

	var apiResp apiResponse
	if err := json.NewDecoder(respBody).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if apiResp.Error != nil {
		return nil, fmt.Errorf("zai API error: %s: %s", apiResp.Error.ErrorCode(), apiResp.Error.Message)
	}

	return z.parseResponse(&apiResp), nil
}

func (z *ZAI) StreamChat(ctx context.Context, req provider.ChatRequest) (<-chan provider.StreamEvent, error) {
	body := z.buildRequest(req, true)

	respBody, err := z.doRequest(ctx, body)
	if err != nil {
		return nil, err
	}

	ch := make(chan provider.StreamEvent, 64)
	go z.parseSSE(respBody, ch)
	return ch, nil
}

func (z *ZAI) buildRequest(req provider.ChatRequest, stream bool) map[string]any {
	body := map[string]any{
		"model": req.Model,
	}
	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	}
	if stream {
		body["stream"] = true
	}

	var msgs []map[string]any
	if sysText := req.FullSystemText(); sysText != "" {
		msgs = append(msgs, map[string]any{
			"role":    "system",
			"content": sysText,
		})
	}
	for _, m := range req.Messages {
		msgs = append(msgs, convertMessage(m)...)
	}
	body["messages"] = msgs

	if len(req.Tools) > 0 {
		var tools []map[string]any
		for _, t := range req.Tools {
			tools = append(tools, map[string]any{
				"type": "function",
				"function": map[string]any{
					"name":        t.Name,
					"description": t.Description,
					"parameters":  json.RawMessage(t.InputSchema),
				},
			})
		}
		body["tools"] = tools
	}

	return body
}

func convertMessage(m provider.Message) []map[string]any {
	if m.Role == provider.RoleUser {
		hasToolResults := false
		for _, block := range m.Content {
			if block.Type == provider.ContentToolResult {
				hasToolResults = true
				break
			}
		}
		if hasToolResults {
			var msgs []map[string]any
			for _, block := range m.Content {
				if block.Type == provider.ContentToolResult && block.ToolResult != nil {
					msgs = append(msgs, map[string]any{
						"role":         "tool",
						"tool_call_id": block.ToolResult.ToolUseID,
						"content":      block.ToolResult.Content,
					})
				}
			}
			return msgs
		}
	}

	if m.Role == provider.RoleAssistant {
		msg := map[string]any{"role": "assistant"}

		var textParts []string
		var toolCalls []map[string]any

		for _, block := range m.Content {
			switch block.Type {
			case provider.ContentText:
				textParts = append(textParts, block.Text)
			case provider.ContentToolUse:
				if block.ToolCall != nil {
					toolCalls = append(toolCalls, map[string]any{
						"id":   block.ToolCall.ID,
						"type": "function",
						"function": map[string]any{
							"name":      block.ToolCall.Name,
							"arguments": string(block.ToolCall.Input),
						},
					})
				}
			}
		}

		if len(textParts) > 0 {
			msg["content"] = strings.Join(textParts, "")
		}
		if len(toolCalls) > 0 {
			msg["tool_calls"] = toolCalls
		}
		return []map[string]any{msg}
	}

	var b strings.Builder
	for _, block := range m.Content {
		if block.Type == provider.ContentText {
			b.WriteString(block.Text)
		}
	}
	return []map[string]any{{
		"role":    string(m.Role),
		"content": b.String(),
	}}
}

func (z *ZAI) doRequest(ctx context.Context, body map[string]any) (io.ReadCloser, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", z.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+z.apiKey)

	resp, err := z.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		var apiErr apiResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err == nil && apiErr.Error != nil {
			return nil, fmt.Errorf("zai API error (HTTP %d): %s: %s", resp.StatusCode, apiErr.Error.ErrorCode(), apiErr.Error.Message)
		}
		return nil, fmt.Errorf("zai API error: HTTP %d", resp.StatusCode)
	}

	return resp.Body, nil
}

func (z *ZAI) parseResponse(resp *apiResponse) *provider.ChatResponse {
	result := &provider.ChatResponse{
		Usage: provider.Usage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
		},
	}

	if len(resp.Choices) == 0 {
		result.StopReason = provider.StopEndTurn
		return result
	}

	choice := resp.Choices[0]

	switch choice.FinishReason {
	case "stop":
		result.StopReason = provider.StopEndTurn
	case "tool_calls":
		result.StopReason = provider.StopToolUse
	case "length":
		result.StopReason = provider.StopMaxTokens
	default:
		result.StopReason = provider.StopEndTurn
	}

	if text := choice.Message.EffectiveContent(); text != "" {
		result.Content = append(result.Content, provider.ContentBlock{
			Type: provider.ContentText,
			Text: text,
		})
	}

	for _, tc := range choice.Message.ToolCalls {
		result.Content = append(result.Content, provider.ContentBlock{
			Type: provider.ContentToolUse,
			ToolCall: &provider.ToolCall{
				ID:    tc.ID,
				Name:  tc.Function.Name,
				Input: json.RawMessage(tc.Function.Arguments),
			},
		})
	}

	return result
}

func (z *ZAI) parseSSE(body io.ReadCloser, ch chan<- provider.StreamEvent) {
	defer close(ch)
	defer body.Close()

	scanner := bufio.NewScanner(body)
	currentToolCalls := make(map[int]*provider.ToolCall)
	currentDeltas := make(map[int]string)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			ch <- provider.StreamEvent{Type: provider.EventDone}
			return
		}

		var event sseChunk
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		if len(event.Choices) == 0 {
			continue
		}

		delta := event.Choices[0].Delta
		finishReason := event.Choices[0].FinishReason

		if text := delta.EffectiveContent(); text != "" {
			ch <- provider.StreamEvent{
				Type: provider.EventTextDelta,
				Text: text,
			}
		}

		for _, tc := range delta.ToolCalls {
			if tc.Function.Name != "" {
				toolCall := &provider.ToolCall{
					ID:   tc.ID,
					Name: tc.Function.Name,
				}
				currentToolCalls[tc.Index] = toolCall
				ch <- provider.StreamEvent{
					Type:     provider.EventToolCallStart,
					ToolCall: toolCall,
				}
			}
			if tc.Function.Arguments != "" {
				currentDeltas[tc.Index] += tc.Function.Arguments
				ch <- provider.StreamEvent{
					Type:  provider.EventToolCallDelta,
					Delta: tc.Function.Arguments,
				}
			}
		}

		if finishReason == "tool_calls" || finishReason == "stop" {
			for idx := range currentToolCalls {
				if _, ok := currentDeltas[idx]; ok {
					ch <- provider.StreamEvent{Type: provider.EventToolCallEnd}
				}
			}
			ch <- provider.StreamEvent{Type: provider.EventDone}
			return
		}
	}

	if err := scanner.Err(); err != nil {
		ch <- provider.StreamEvent{
			Type:  provider.EventError,
			Error: fmt.Errorf("stream read error: %w", err),
		}
	}
}

type apiResponse struct {
	Choices []apiChoice `json:"choices"`
	Usage   apiUsage    `json:"usage"`
	Error   *apiError   `json:"error,omitempty"`
}

type apiChoice struct {
	Message      apiMessage `json:"message"`
	FinishReason string     `json:"finish_reason"`
}

type apiMessage struct {
	Role             string        `json:"role"`
	Content          string        `json:"content"`
	ReasoningContent string        `json:"reasoning_content,omitempty"`
	ToolCalls        []apiToolCall `json:"tool_calls,omitempty"`
}

func (m *apiMessage) EffectiveContent() string {
	if m.Content != "" {
		return m.Content
	}
	return m.ReasoningContent
}

type apiToolCall struct {
	ID       string      `json:"id"`
	Type     string      `json:"type"`
	Function apiFunction `json:"function"`
}

type apiFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type apiUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

type apiError struct {
	Type    string `json:"type"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *apiError) ErrorCode() string {
	if e.Type != "" {
		return e.Type
	}
	return e.Code
}

type sseChunk struct {
	Choices []sseChoice `json:"choices"`
}

type sseChoice struct {
	Delta        sseDelta `json:"delta"`
	FinishReason string   `json:"finish_reason"`
}

type sseDelta struct {
	Content          string        `json:"content"`
	ReasoningContent string        `json:"reasoning_content,omitempty"`
	ToolCalls        []sseToolCall `json:"tool_calls,omitempty"`
}

func (d *sseDelta) EffectiveContent() string {
	if d.Content != "" {
		return d.Content
	}
	return d.ReasoningContent
}

type sseToolCall struct {
	Index    int         `json:"index"`
	ID       string      `json:"id"`
	Function apiFunction `json:"function"`
}
