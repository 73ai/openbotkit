package provider

import (
	"context"
	"fmt"
	"testing"
	"time"
)

type mockRetryProvider struct {
	errors []error
	calls  int
}

func (m *mockRetryProvider) Chat(_ context.Context, _ ChatRequest) (*ChatResponse, error) {
	m.calls++
	if m.calls <= len(m.errors) {
		return nil, m.errors[m.calls-1]
	}
	return &ChatResponse{StopReason: StopEndTurn}, nil
}

func (m *mockRetryProvider) StreamChat(_ context.Context, _ ChatRequest) (<-chan StreamEvent, error) {
	m.calls++
	if m.calls <= len(m.errors) {
		return nil, m.errors[m.calls-1]
	}
	ch := make(chan StreamEvent)
	close(ch)
	return ch, nil
}

func TestResilientProvider_RetryOn429(t *testing.T) {
	mock := &mockRetryProvider{
		errors: []error{
			fmt.Errorf("API error (HTTP 429): rate limit"),
			fmt.Errorf("API error (HTTP 429): rate limit"),
		},
	}
	rp := &ResilientProvider{inner: mock, maxRetries: 3, baseDelay: 1 * time.Millisecond}

	resp, err := rp.Chat(context.Background(), ChatRequest{})
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if mock.calls != 3 {
		t.Errorf("expected 3 calls, got %d", mock.calls)
	}
}

func TestResilientProvider_NoRetryOn401(t *testing.T) {
	mock := &mockRetryProvider{
		errors: []error{
			fmt.Errorf("API error (HTTP 401): unauthorized"),
		},
	}
	rp := &ResilientProvider{inner: mock, maxRetries: 3, baseDelay: 1 * time.Millisecond}

	_, err := rp.Chat(context.Background(), ChatRequest{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if mock.calls != 1 {
		t.Errorf("expected 1 call (no retry), got %d", mock.calls)
	}
}

func TestResilientProvider_MaxRetriesExhausted(t *testing.T) {
	mock := &mockRetryProvider{
		errors: []error{
			fmt.Errorf("API error (HTTP 500): server error"),
			fmt.Errorf("API error (HTTP 500): server error"),
			fmt.Errorf("API error (HTTP 500): server error"),
		},
	}
	rp := &ResilientProvider{inner: mock, maxRetries: 3, baseDelay: 1 * time.Millisecond}

	_, err := rp.Chat(context.Background(), ChatRequest{})
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if mock.calls != 3 {
		t.Errorf("expected 3 calls, got %d", mock.calls)
	}
}

func TestResilientProvider_StreamRetry(t *testing.T) {
	mock := &mockRetryProvider{
		errors: []error{
			fmt.Errorf("API error (HTTP 429): rate limit"),
		},
	}
	rp := &ResilientProvider{inner: mock, maxRetries: 3, baseDelay: 1 * time.Millisecond}

	ch, err := rp.StreamChat(context.Background(), ChatRequest{})
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if ch == nil {
		t.Fatal("expected channel, got nil")
	}
	if mock.calls != 2 {
		t.Errorf("expected 2 calls, got %d", mock.calls)
	}
}
