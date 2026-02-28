package gmail

import (
	"context"
	"fmt"
	"net/http"
	"testing"
)

type mockProvider struct {
	accounts []string
	err      error
}

func (m *mockProvider) Accounts(ctx context.Context) ([]string, error) {
	return m.accounts, m.err
}

func (m *mockProvider) Client(ctx context.Context, account string, scopes []string) (*http.Client, error) {
	return nil, fmt.Errorf("not implemented in mock")
}

func TestResolveAccount(t *testing.T) {
	tests := []struct {
		name     string
		accounts []string
		accErr   error
		input    string
		want     string
		wantErr  string
	}{
		{
			name:     "single account auto-selected",
			accounts: []string{"alice@example.com"},
			input:    "",
			want:     "alice@example.com",
		},
		{
			name:     "explicit account found",
			accounts: []string{"alice@example.com", "bob@example.com"},
			input:    "bob@example.com",
			want:     "bob@example.com",
		},
		{
			name:     "explicit account not found",
			accounts: []string{"alice@example.com"},
			input:    "unknown@example.com",
			wantErr:  "not authenticated",
		},
		{
			name:     "multiple accounts without selection",
			accounts: []string{"alice@example.com", "bob@example.com"},
			input:    "",
			wantErr:  "multiple accounts found",
		},
		{
			name:     "no accounts",
			accounts: []string{},
			input:    "",
			wantErr:  "no authenticated accounts",
		},
		{
			name:    "provider error",
			accErr:  fmt.Errorf("db error"),
			input:   "",
			wantErr: "list accounts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := New(Config{Provider: &mockProvider{
				accounts: tt.accounts,
				err:      tt.accErr,
			}})

			got, err := g.resolveAccount(context.Background(), tt.input)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !contains(err.Error(), tt.wantErr) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
