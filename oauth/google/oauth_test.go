package google

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMergeScopes(t *testing.T) {
	tests := []struct {
		name string
		a, b []string
		want []string
	}{
		{
			name: "no overlap",
			a:    []string{"a", "b"},
			b:    []string{"c", "d"},
			want: []string{"a", "b", "c", "d"},
		},
		{
			name: "with overlap",
			a:    []string{"a", "b", "c"},
			b:    []string{"b", "c", "d"},
			want: []string{"a", "b", "c", "d"},
		},
		{
			name: "empty a",
			a:    nil,
			b:    []string{"x"},
			want: []string{"x"},
		},
		{
			name: "empty b",
			a:    []string{"x"},
			b:    nil,
			want: []string{"x"},
		},
		{
			name: "both empty",
			a:    nil,
			b:    nil,
			want: nil,
		},
		{
			name: "duplicates within a",
			a:    []string{"a", "a", "b"},
			b:    []string{"b"},
			want: []string{"a", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeScopes(tt.a, tt.b)
			if len(got) != len(tt.want) {
				t.Fatalf("len: got %d, want %d (%v vs %v)", len(got), len(tt.want), got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("index %d: got %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// fakeCredentials returns a minimal Google OAuth credentials JSON for testing.
func fakeCredentials(t *testing.T) string {
	t.Helper()
	cred := `{"installed":{"client_id":"test.apps.googleusercontent.com","client_secret":"secret","redirect_uris":["http://localhost"]}}`
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.json")
	if err := os.WriteFile(path, []byte(cred), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadConfig_DefaultRedirectURL(t *testing.T) {
	path := fakeCredentials(t)
	cfg, err := loadConfig(path, []string{"openid"}, "")
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.RedirectURL != "http://localhost:8085/callback" {
		t.Fatalf("expected default redirect, got %q", cfg.RedirectURL)
	}
}

func TestLoadConfig_CustomRedirectURL(t *testing.T) {
	path := fakeCredentials(t)
	want := "https://example.ngrok-free.app/auth/google/callback"
	cfg, err := loadConfig(path, []string{"openid"}, want)
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.RedirectURL != want {
		t.Fatalf("got %q, want %q", cfg.RedirectURL, want)
	}
}

func TestLoadConfig_MergesImplicitScopes(t *testing.T) {
	path := fakeCredentials(t)
	cfg, err := loadConfig(path, []string{"https://www.googleapis.com/auth/drive"}, "")
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	scopeSet := make(map[string]bool)
	for _, s := range cfg.Scopes {
		scopeSet[s] = true
	}
	for _, want := range []string{"openid", "email", "https://www.googleapis.com/auth/drive"} {
		if !scopeSet[want] {
			t.Errorf("missing scope %q in %v", want, cfg.Scopes)
		}
	}
}

func TestLoadConfig_MissingFile(t *testing.T) {
	_, err := loadConfig("/nonexistent/credentials.json", []string{"openid"}, "")
	if err == nil {
		t.Fatal("expected error for missing credentials file")
	}
}
