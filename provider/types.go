package provider

import "encoding/json"

// Role represents a message participant.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
)

// StopReason indicates why the model stopped generating.
type StopReason string

const (
	StopEndTurn   StopReason = "end_turn"
	StopToolUse   StopReason = "tool_use"
	StopMaxTokens StopReason = "max_tokens"
)

// ContentBlockType identifies the kind of content block.
type ContentBlockType string

const (
	ContentText       ContentBlockType = "text"
	ContentToolUse    ContentBlockType = "tool_use"
	ContentToolResult ContentBlockType = "tool_result"
)

// Message represents a single message in the conversation.
type Message struct {
	Role    Role           `json:"role"`
	Content []ContentBlock `json:"content"`
}

// NewTextMessage creates a message with a single text block.
func NewTextMessage(role Role, text string) Message {
	return Message{
		Role:    role,
		Content: []ContentBlock{{Type: ContentText, Text: text}},
	}
}

// ContentBlock is a polymorphic content element within a message.
// Only one of Text, ToolCall, or ToolResult is populated based on Type.
type ContentBlock struct {
	Type       ContentBlockType `json:"type"`
	Text       string           `json:"text,omitempty"`
	ToolCall   *ToolCall        `json:"tool_call,omitempty"`
	ToolResult *ToolResult      `json:"tool_result,omitempty"`
}

// ToolCall represents a model's request to invoke a tool.
type ToolCall struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// ToolResult represents the output of a tool invocation.
type ToolResult struct {
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error,omitempty"`
}

// Tool defines a tool that can be offered to the model.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// ChatRequest is the input to Provider.Chat and Provider.StreamChat.
type ChatRequest struct {
	Model     string    `json:"model"`
	System    string    `json:"system,omitempty"`
	Messages  []Message `json:"messages"`
	Tools     []Tool    `json:"tools,omitempty"`
	MaxTokens int       `json:"max_tokens,omitempty"`
}

// ChatResponse is the output of Provider.Chat.
type ChatResponse struct {
	Content    []ContentBlock `json:"content"`
	StopReason StopReason     `json:"stop_reason"`
	Usage      Usage          `json:"usage"`
}

// Usage tracks token consumption.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// StreamEventType identifies the kind of streaming event.
type StreamEventType string

const (
	EventTextDelta     StreamEventType = "text_delta"
	EventToolCallStart StreamEventType = "tool_call_start"
	EventToolCallDelta StreamEventType = "tool_call_delta"
	EventToolCallEnd   StreamEventType = "tool_call_end"
	EventDone          StreamEventType = "done"
	EventError         StreamEventType = "error"
)

// StreamEvent represents a single streaming event from the model.
type StreamEvent struct {
	Type StreamEventType `json:"type"`

	// For EventTextDelta.
	Text string `json:"text,omitempty"`

	// For EventToolCallStart.
	ToolCall *ToolCall `json:"tool_call,omitempty"`

	// For EventToolCallDelta — partial JSON input.
	Delta string `json:"delta,omitempty"`

	// For EventDone.
	Response *ChatResponse `json:"response,omitempty"`

	// For EventError.
	Error error `json:"-"`
}

// TextContent returns the concatenated text from all text content blocks.
func (r *ChatResponse) TextContent() string {
	var result string
	for _, block := range r.Content {
		if block.Type == ContentText {
			result += block.Text
		}
	}
	return result
}

// ToolCalls returns all tool call content blocks from the response.
func (r *ChatResponse) ToolCalls() []ToolCall {
	var calls []ToolCall
	for _, block := range r.Content {
		if block.Type == ContentToolUse && block.ToolCall != nil {
			calls = append(calls, *block.ToolCall)
		}
	}
	return calls
}
