package provider

import (
	"fmt"
	"testing"
)

func TestAPIError_Error(t *testing.T) {
	e := &APIError{StatusCode: 429, Kind: ErrorRetryable, Message: "rate limit"}
	got := e.Error()
	want := "API error (HTTP 429): rate limit"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestErrorPermanent_IsZeroValue(t *testing.T) {
	var k ErrorKind
	if k != ErrorPermanent {
		t.Errorf("zero-value ErrorKind = %d, want ErrorPermanent (%d)", k, ErrorPermanent)
	}
	// A default-initialized APIError should also be permanent.
	e := &APIError{}
	if e.Kind != ErrorPermanent {
		t.Errorf("default APIError.Kind = %d, want ErrorPermanent (%d)", e.Kind, ErrorPermanent)
	}
}

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantKind   ErrorKind
		wantStatus int
	}{
		{
			name:       "429 rate limit",
			err:        fmt.Errorf("anthropic API error (HTTP 429): rate_limit_error: Too many requests"),
			wantKind:   ErrorRetryable,
			wantStatus: 429,
		},
		{
			name:       "500 server error",
			err:        fmt.Errorf("API error (HTTP 500): internal server error"),
			wantKind:   ErrorRetryable,
			wantStatus: 500,
		},
		{
			name:       "503 service unavailable",
			err:        fmt.Errorf("API error (HTTP 503): service unavailable"),
			wantKind:   ErrorRetryable,
			wantStatus: 503,
		},
		{
			name:       "401 unauthorized",
			err:        fmt.Errorf("API error (HTTP 401): invalid API key"),
			wantKind:   ErrorAuth,
			wantStatus: 401,
		},
		{
			name:       "403 forbidden",
			err:        fmt.Errorf("API error (HTTP 403): forbidden"),
			wantKind:   ErrorAuth,
			wantStatus: 403,
		},
		{
			name:       "400 context window",
			err:        fmt.Errorf("API error (HTTP 400): context length exceeded"),
			wantKind:   ErrorContextWindow,
			wantStatus: 400,
		},
		{
			name:       "400 other",
			err:        fmt.Errorf("API error (HTTP 400): invalid request"),
			wantKind:   ErrorPermanent,
			wantStatus: 400,
		},
		{
			name:       "502 bad gateway",
			err:        fmt.Errorf("API error (HTTP 502): bad gateway"),
			wantKind:   ErrorRetryable,
			wantStatus: 502,
		},
		{
			name:       "404 not found",
			err:        fmt.Errorf("API error (HTTP 404): not found"),
			wantKind:   ErrorPermanent,
			wantStatus: 404,
		},
		{
			name:       "500 with context in message",
			err:        fmt.Errorf("API error (HTTP 500): internal context error"),
			wantKind:   ErrorRetryable,
			wantStatus: 500,
		},
		{
			name:       "400 context uppercase",
			err:        fmt.Errorf("API error (HTTP 400): CONTEXT window too large"),
			wantKind:   ErrorContextWindow,
			wantStatus: 400,
		},
		{
			name:       "multiple status codes picks first",
			err:        fmt.Errorf("API error (HTTP 429): retry after (HTTP 500)"),
			wantKind:   ErrorRetryable,
			wantStatus: 429,
		},
		{
			name:       "no status code",
			err:        fmt.Errorf("connection refused"),
			wantKind:   ErrorPermanent,
			wantStatus: 0,
		},
		{
			name:       "nil error",
			err:        nil,
			wantKind:   0,
			wantStatus: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiErr := ClassifyError(tt.err)
			if tt.err == nil {
				if apiErr != nil {
					t.Fatalf("expected nil, got %v", apiErr)
				}
				return
			}
			if apiErr.Kind != tt.wantKind {
				t.Errorf("Kind = %d, want %d", apiErr.Kind, tt.wantKind)
			}
			if apiErr.StatusCode != tt.wantStatus {
				t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, tt.wantStatus)
			}
		})
	}
}
