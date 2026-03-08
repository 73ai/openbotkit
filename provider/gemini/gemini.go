package gemini

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/priyanshujain/openbotkit/provider"
)

const defaultBaseURL = "https://generativelanguage.googleapis.com"

// Gemini implements provider.Provider using the Google Gemini API.
// It supports both the direct Gemini API and Google Vertex AI.
type Gemini struct {
	apiKey  string
	baseURL string
	client  *http.Client

	// Vertex AI fields.
	vertexProject string
	vertexRegion  string
	tokenOnce     sync.Once
	tokenSource   oauth2.TokenSource
}

var _ provider.Provider = (*Gemini)(nil)

// Option configures the Gemini provider.
type Option func(*Gemini)

func WithBaseURL(url string) Option {
	return func(g *Gemini) { g.baseURL = url }
}

func WithHTTPClient(c *http.Client) Option {
	return func(g *Gemini) { g.client = c }
}

// WithVertexAI configures the provider to use Google Vertex AI.
// Uses Google Application Default Credentials unless a custom TokenSource is provided via WithTokenSource.
func WithVertexAI(project, region string) Option {
	return func(g *Gemini) {
		g.vertexProject = project
		g.vertexRegion = region
	}
}

// WithTokenSource sets a custom OAuth2 token source for Vertex AI authentication.
func WithTokenSource(ts oauth2.TokenSource) Option {
	return func(g *Gemini) { g.tokenSource = ts }
}

func (g *Gemini) isVertexAI() bool {
	return g.vertexProject != ""
}

func (g *Gemini) vertexToken(ctx context.Context) (string, error) {
	if g.tokenSource == nil {
		var initErr error
		g.tokenOnce.Do(func() {
			g.tokenSource, initErr = google.DefaultTokenSource(ctx, "https://www.googleapis.com/auth/cloud-platform")
		})
		if initErr != nil {
			return "", fmt.Errorf("get google credentials: %w", initErr)
		}
	}
	token, err := g.tokenSource.Token()
	if err != nil {
		return "", fmt.Errorf("get access token: %w", err)
	}
	return token.AccessToken, nil
}

func New(apiKey string, opts ...Option) *Gemini {
	g := &Gemini{
		apiKey:  apiKey,
		baseURL: defaultBaseURL,
		client:  http.DefaultClient,
	}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

func init() {
	provider.RegisterFactory("gemini", func(apiKey, baseURL string) provider.Provider {
		var opts []Option
		if baseURL != "" {
			opts = append(opts, WithBaseURL(baseURL))
		}
		return New(apiKey, opts...)
	})
}

// Chat sends a non-streaming request.
func (g *Gemini) Chat(ctx context.Context, req provider.ChatRequest) (*provider.ChatResponse, error) {
	body := g.buildRequest(req)

	url, err := g.chatURL(ctx, req.Model, false)
	if err != nil {
		return nil, err
	}
	respBody, err := g.doRequest(ctx, url, body)
	if err != nil {
		return nil, err
	}
	defer respBody.Close()

	var apiResp apiResponse
	if err := json.NewDecoder(respBody).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if apiResp.Error != nil {
		return nil, fmt.Errorf("gemini API error: %d: %s", apiResp.Error.Code, apiResp.Error.Message)
	}

	return g.parseResponse(&apiResp), nil
}

// StreamChat sends a streaming request.
func (g *Gemini) StreamChat(ctx context.Context, req provider.ChatRequest) (<-chan provider.StreamEvent, error) {
	body := g.buildRequest(req)

	url, err := g.chatURL(ctx, req.Model, true)
	if err != nil {
		return nil, err
	}
	respBody, err := g.doRequest(ctx, url, body)
	if err != nil {
		return nil, err
	}

	ch := make(chan provider.StreamEvent, 64)
	go g.parseSSE(respBody, ch)
	return ch, nil
}

// chatURL returns the endpoint URL, handling both direct API and Vertex AI.
func (g *Gemini) chatURL(ctx context.Context, model string, stream bool) (string, error) {
	if g.isVertexAI() {
		action := "generateContent"
		suffix := ""
		if stream {
			action = "streamGenerateContent"
			suffix = "?alt=sse"
		}
		return fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/google/models/%s:%s%s",
			g.vertexRegion, g.vertexProject, g.vertexRegion, model, action, suffix), nil
	}

	if stream {
		return fmt.Sprintf("%s/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s", g.baseURL, model, g.apiKey), nil
	}
	return fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", g.baseURL, model, g.apiKey), nil
}

func (g *Gemini) buildRequest(req provider.ChatRequest) map[string]any {
	body := map[string]any{}

	if req.System != "" {
		body["systemInstruction"] = map[string]any{
			"parts": []map[string]any{{"text": req.System}},
		}
	}

	var contents []map[string]any
	for _, m := range req.Messages {
		contents = append(contents, convertMessage(m)...)
	}
	body["contents"] = contents

	if len(req.Tools) > 0 {
		var decls []map[string]any
		for _, t := range req.Tools {
			decl := map[string]any{
				"name":        t.Name,
				"description": t.Description,
			}
			// Parse InputSchema to extract parameters (Gemini wants the schema object directly).
			var schema map[string]any
			if err := json.Unmarshal(t.InputSchema, &schema); err == nil {
				decl["parameters"] = schema
			}
			decls = append(decls, decl)
		}
		body["tools"] = []map[string]any{{"functionDeclarations": decls}}
	}

	if req.MaxTokens > 0 {
		body["generationConfig"] = map[string]any{
			"maxOutputTokens": req.MaxTokens,
		}
	}

	return body
}

// convertMessage translates a provider.Message into Gemini content(s).
func convertMessage(m provider.Message) []map[string]any {
	role := string(m.Role)
	if role == "assistant" {
		role = "model"
	}

	var parts []map[string]any
	for _, block := range m.Content {
		switch block.Type {
		case provider.ContentText:
			parts = append(parts, map[string]any{"text": block.Text})
		case provider.ContentToolUse:
			if block.ToolCall != nil {
				var args map[string]any
				_ = json.Unmarshal(block.ToolCall.Input, &args)
				parts = append(parts, map[string]any{
					"functionCall": map[string]any{
						"name": block.ToolCall.Name,
						"args": args,
					},
				})
			}
		case provider.ContentToolResult:
			if block.ToolResult != nil {
				// Gemini expects functionResponse in a separate "user" content.
				var response map[string]any
				if err := json.Unmarshal([]byte(block.ToolResult.Content), &response); err != nil {
					response = map[string]any{"result": block.ToolResult.Content}
				}
				return []map[string]any{{
					"role": "user",
					"parts": []map[string]any{{
						"functionResponse": map[string]any{
							"name":     block.ToolResult.ToolUseID,
							"response": response,
						},
					}},
				}}
			}
		}
	}

	return []map[string]any{{
		"role":  role,
		"parts": parts,
	}}
}

func (g *Gemini) doRequest(ctx context.Context, url string, body map[string]any) (io.ReadCloser, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	if g.isVertexAI() {
		token, err := g.vertexToken(ctx)
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		var apiErr apiResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err == nil && apiErr.Error != nil {
			return nil, fmt.Errorf("gemini API error (HTTP %d): %s", resp.StatusCode, apiErr.Error.Message)
		}
		return nil, fmt.Errorf("gemini API error: HTTP %d", resp.StatusCode)
	}

	return resp.Body, nil
}

func (g *Gemini) parseResponse(resp *apiResponse) *provider.ChatResponse {
	result := &provider.ChatResponse{
		Usage: provider.Usage{
			InputTokens:  resp.UsageMetadata.PromptTokenCount,
			OutputTokens: resp.UsageMetadata.CandidatesTokenCount,
		},
	}

	if len(resp.Candidates) == 0 {
		result.StopReason = provider.StopEndTurn
		return result
	}

	candidate := resp.Candidates[0]
	hasToolCalls := false

	for i, part := range candidate.Content.Parts {
		if part.Text != "" {
			result.Content = append(result.Content, provider.ContentBlock{
				Type: provider.ContentText,
				Text: part.Text,
			})
		}
		if part.FunctionCall != nil {
			hasToolCalls = true
			argsJSON, _ := json.Marshal(part.FunctionCall.Args)
			result.Content = append(result.Content, provider.ContentBlock{
				Type: provider.ContentToolUse,
				ToolCall: &provider.ToolCall{
					ID:    fmt.Sprintf("call_%d", i),
					Name:  part.FunctionCall.Name,
					Input: argsJSON,
				},
			})
		}
	}

	if hasToolCalls {
		result.StopReason = provider.StopToolUse
	} else {
		switch candidate.FinishReason {
		case "MAX_TOKENS":
			result.StopReason = provider.StopMaxTokens
		default:
			result.StopReason = provider.StopEndTurn
		}
	}

	return result
}

func (g *Gemini) parseSSE(body io.ReadCloser, ch chan<- provider.StreamEvent) {
	defer close(ch)
	defer body.Close()

	scanner := bufio.NewScanner(body)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		var resp apiResponse
		if err := json.Unmarshal([]byte(data), &resp); err != nil {
			continue
		}

		if resp.Error != nil {
			ch <- provider.StreamEvent{
				Type:  provider.EventError,
				Error: fmt.Errorf("stream error: %s", resp.Error.Message),
			}
			return
		}

		if len(resp.Candidates) == 0 {
			continue
		}

		candidate := resp.Candidates[0]
		for i, part := range candidate.Content.Parts {
			if part.Text != "" {
				ch <- provider.StreamEvent{
					Type: provider.EventTextDelta,
					Text: part.Text,
				}
			}
			if part.FunctionCall != nil {
				argsJSON, _ := json.Marshal(part.FunctionCall.Args)
				toolCall := &provider.ToolCall{
					ID:   fmt.Sprintf("call_%d", i),
					Name: part.FunctionCall.Name,
				}
				ch <- provider.StreamEvent{
					Type:     provider.EventToolCallStart,
					ToolCall: toolCall,
				}
				ch <- provider.StreamEvent{
					Type:  provider.EventToolCallDelta,
					Delta: string(argsJSON),
				}
				ch <- provider.StreamEvent{
					Type: provider.EventToolCallEnd,
				}
			}
		}

		if candidate.FinishReason == "STOP" || candidate.FinishReason == "MAX_TOKENS" {
			ch <- provider.StreamEvent{Type: provider.EventDone}
			return
		}
	}
}

// API types for JSON marshaling.

type apiResponse struct {
	Candidates    []apiCandidate `json:"candidates"`
	UsageMetadata apiUsage       `json:"usageMetadata"`
	Error         *apiError      `json:"error,omitempty"`
}

type apiCandidate struct {
	Content      apiContent `json:"content"`
	FinishReason string     `json:"finishReason"`
}

type apiContent struct {
	Role  string    `json:"role"`
	Parts []apiPart `json:"parts"`
}

type apiPart struct {
	Text         string        `json:"text,omitempty"`
	FunctionCall *apiFuncCall  `json:"functionCall,omitempty"`
}

type apiFuncCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

type apiUsage struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
}

type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
