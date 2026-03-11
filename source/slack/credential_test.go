package slack

import (
	"testing"

	"github.com/zalando/go-keyring"
)

func TestValidateToken(t *testing.T) {
	tests := []struct {
		token   string
		wantErr bool
	}{
		{"xoxp-123", false},
		{"xoxb-abc", false},
		{"xoxc-def", false},
		{"", true},
		{"invalid", true},
		{"xoxa-123", true},
	}
	for _, tt := range tests {
		err := ValidateToken(tt.token)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateToken(%q) err=%v, wantErr=%v", tt.token, err, tt.wantErr)
		}
	}
}

func TestTokenAccount(t *testing.T) {
	got := tokenAccount("mywork")
	want := "slack/mywork/token"
	if got != want {
		t.Errorf("tokenAccount = %q, want %q", got, want)
	}
}

func TestCookieAccount(t *testing.T) {
	got := cookieAccount("mywork")
	want := "slack/mywork/cookie"
	if got != want {
		t.Errorf("cookieAccount = %q, want %q", got, want)
	}
}

func TestSaveLoadDeleteCredentials(t *testing.T) {
	keyring.MockInit()

	err := SaveCredentials("testws", "xoxp-token123", "xoxd-cookie456")
	if err != nil {
		t.Fatalf("SaveCredentials: %v", err)
	}

	creds, err := LoadCredentials("testws")
	if err != nil {
		t.Fatalf("LoadCredentials: %v", err)
	}
	if creds.Token != "xoxp-token123" {
		t.Errorf("token = %q", creds.Token)
	}
	if creds.Cookie != "xoxd-cookie456" {
		t.Errorf("cookie = %q", creds.Cookie)
	}

	err = DeleteCredentials("testws")
	if err != nil {
		t.Fatalf("DeleteCredentials: %v", err)
	}

	_, err = LoadCredentials("testws")
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestSaveCredentials_NoCookie(t *testing.T) {
	keyring.MockInit()

	err := SaveCredentials("ws2", "xoxb-token", "")
	if err != nil {
		t.Fatalf("SaveCredentials: %v", err)
	}

	creds, err := LoadCredentials("ws2")
	if err != nil {
		t.Fatalf("LoadCredentials: %v", err)
	}
	if creds.Token != "xoxb-token" {
		t.Errorf("token = %q", creds.Token)
	}
	if creds.Cookie != "" {
		t.Errorf("cookie should be empty, got %q", creds.Cookie)
	}
}

func TestLoadCredentials_NotFound(t *testing.T) {
	keyring.MockInit()

	_, err := LoadCredentials("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent workspace")
	}
}

func TestListWorkspaces(t *testing.T) {
	_, err := ListWorkspaces()
	if err == nil {
		t.Error("expected error from ListWorkspaces")
	}
}
