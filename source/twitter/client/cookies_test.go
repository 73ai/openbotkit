package client

import (
	"testing"
)

func TestNewSessionWithCSRF(t *testing.T) {
	s := NewSessionWithCSRF("auth123", "csrf456")
	if s.AuthToken != "auth123" {
		t.Errorf("AuthToken = %q, want %q", s.AuthToken, "auth123")
	}
	if s.CSRFToken != "csrf456" {
		t.Errorf("CSRFToken = %q, want %q", s.CSRFToken, "csrf456")
	}
}

func TestNewSessionWithCSRF_EmptyCSRF(t *testing.T) {
	s := NewSessionWithCSRF("auth123", "")
	if s.AuthToken != "auth123" {
		t.Errorf("AuthToken = %q, want %q", s.AuthToken, "auth123")
	}
	if s.CSRFToken == "" {
		t.Error("CSRFToken should be auto-generated when empty")
	}
	if len(s.CSRFToken) != 32 {
		t.Errorf("CSRFToken length = %d, want 32 (hex-encoded 16 bytes)", len(s.CSRFToken))
	}
}
