package slack

import "testing"

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
