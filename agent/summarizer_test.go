package agent

import (
	"context"
	"fmt"
	"testing"

	"github.com/priyanshujain/openbotkit/provider"
)

type mockSummarizerProvider struct {
	resp *provider.ChatResponse
	err  error
	reqs []provider.ChatRequest
}

func (m *mockSummarizerProvider) Chat(_ context.Context, req provider.ChatRequest) (*provider.ChatResponse, error) {
	m.reqs = append(m.reqs, req)
	return m.resp, m.err
}

func (m *mockSummarizerProvider) StreamChat(_ context.Context, req provider.ChatRequest) (<-chan provider.StreamEvent, error) {
	return nil, fmt.Errorf("not implemented")
}

func TestLLMSummarizer_Success(t *testing.T) {
	mp := &mockSummarizerProvider{
		resp: &provider.ChatResponse{
			Content:    []provider.ContentBlock{{Type: provider.ContentText, Text: "User prefers Go. Discussed auth refactor."}},
			StopReason: provider.StopEndTurn,
		},
	}
	s := &LLMSummarizer{Provider: mp, Model: "test-model"}

	msgs := []provider.Message{
		provider.NewTextMessage(provider.RoleUser, "hello"),
		provider.NewTextMessage(provider.RoleAssistant, "hi there"),
		provider.NewTextMessage(provider.RoleUser, "let's refactor auth"),
		provider.NewTextMessage(provider.RoleAssistant, "sure, using Go"),
	}

	result, err := s.Summarize(context.Background(), msgs)
	if err != nil {
		t.Fatalf("Summarize: %v", err)
	}
	if result != "User prefers Go. Discussed auth refactor." {
		t.Errorf("result = %q", result)
	}
	if len(mp.reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(mp.reqs))
	}
	if mp.reqs[0].System != summarizePrompt {
		t.Error("expected summarize system prompt")
	}
}

func TestLLMSummarizer_ProviderError(t *testing.T) {
	mp := &mockSummarizerProvider{err: fmt.Errorf("api error")}
	s := &LLMSummarizer{Provider: mp, Model: "test-model"}

	_, err := s.Summarize(context.Background(), []provider.Message{
		provider.NewTextMessage(provider.RoleUser, "hello"),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLLMSummarizer_EmptyMessages(t *testing.T) {
	mp := &mockSummarizerProvider{
		resp: &provider.ChatResponse{
			Content:    []provider.ContentBlock{{Type: provider.ContentText, Text: "Empty conversation."}},
			StopReason: provider.StopEndTurn,
		},
	}
	s := &LLMSummarizer{Provider: mp, Model: "test-model"}

	result, err := s.Summarize(context.Background(), nil)
	if err != nil {
		t.Fatalf("Summarize: %v", err)
	}
	if result != "Empty conversation." {
		t.Errorf("result = %q", result)
	}
}
