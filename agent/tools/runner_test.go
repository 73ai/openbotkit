package tools

import (
	"context"
	"strings"
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
