package cli

import (
	"os/exec"
	"testing"
)

func TestParseLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"two accounts", "alice@example.com\nbob@example.com\n", []string{"alice@example.com", "bob@example.com"}},
		{"single line", "one-project\n", []string{"one-project"}},
		{"empty", "", nil},
		{"only whitespace", "  \n  \n", nil},
		{"trailing newlines", "a\nb\n\n\n", []string{"a", "b"}},
		{"spaces around lines", "  a  \n  b  \n", []string{"a", "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLines(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("parseLines(%q) = %v, want %v", tt.input, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseLines(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestIsExecNotFound(t *testing.T) {
	// exec.Error is returned when the binary is not found.
	err := &exec.Error{Name: "gcloud", Err: exec.ErrNotFound}
	if !isExecNotFound(err) {
		t.Error("expected true for exec.Error")
	}

	// exec.ExitError is returned when the binary exits non-zero.
	if isExecNotFound(&exec.ExitError{}) {
		t.Error("expected false for exec.ExitError")
	}
}

func TestGcloudAccounts_Live(t *testing.T) {
	if _, err := exec.LookPath("gcloud"); err != nil {
		t.Skip("gcloud not installed")
	}

	accounts, err := gcloudAccounts()
	if err != nil {
		t.Fatalf("gcloudAccounts: %v", err)
	}
	if len(accounts) == 0 {
		t.Skip("no gcloud accounts configured")
	}
	for _, a := range accounts {
		if a == "" {
			t.Error("got empty account string")
		}
	}
}

func TestGcloudProjects_Live(t *testing.T) {
	if _, err := exec.LookPath("gcloud"); err != nil {
		t.Skip("gcloud not installed")
	}

	accounts, err := gcloudAccounts()
	if err != nil {
		t.Fatalf("gcloudAccounts: %v", err)
	}
	if len(accounts) == 0 {
		t.Skip("no gcloud accounts configured")
	}

	projects, err := gcloudProjects("") // empty = use active account
	if err != nil {
		t.Fatalf("gcloudProjects: %v", err)
	}
	if len(projects) == 0 {
		t.Skip("no gcloud projects visible")
	}
	for _, p := range projects {
		if p == "" {
			t.Error("got empty project string")
		}
	}
}
