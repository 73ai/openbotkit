package tools

import (
	"context"
	"os/exec"
	"strings"
	"testing"
)

// mockRunner records executed commands and returns canned output.
type mockRunner struct {
	outputs map[string]string
	ran     []struct {
		args []string
		env  []string
	}
	err error
}

func (m *mockRunner) Run(_ context.Context, args []string, env []string) (string, error) {
	m.ran = append(m.ran, struct {
		args []string
		env  []string
	}{args, env})
	if m.err != nil {
		return "", m.err
	}
	key := strings.Join(args, " ")
	if out, ok := m.outputs[key]; ok {
		return out, nil
	}
	return "ok", nil
}

func TestGWSRunner_VersionCheck(t *testing.T) {
	if _, err := exec.LookPath("gws"); err != nil {
		t.Skip("gws not on PATH")
	}
	r := NewGWSRunner()
	out, err := r.Run(context.Background(), []string{"--version"}, nil)
	if err != nil {
		t.Fatalf("Run --version: %v", err)
	}
	if out == "" {
		t.Error("expected version output")
	}
}

func TestGWSRunner_EnvInjection(t *testing.T) {
	if _, err := exec.LookPath("gws"); err != nil {
		t.Skip("gws not on PATH")
	}
	r := NewGWSRunner()
	// gws --version should work regardless of env vars
	_, err := r.Run(context.Background(), []string{"--version"}, []string{"GOOGLE_WORKSPACE_CLI_TOKEN=test-token"})
	if err != nil {
		t.Fatalf("Run with env: %v", err)
	}
}

func TestMockRunner_CannedOutput(t *testing.T) {
	m := &mockRunner{outputs: map[string]string{"hello world": "greeting"}}
	out, err := m.Run(context.Background(), []string{"hello", "world"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out != "greeting" {
		t.Errorf("output = %q, want greeting", out)
	}
	if len(m.ran) != 1 {
		t.Fatalf("expected 1 run, got %d", len(m.ran))
	}
	if strings.Join(m.ran[0].args, " ") != "hello world" {
		t.Errorf("args = %v", m.ran[0].args)
	}
}
