package tools

import (
	"context"
	"slices"
	"strings"
	"testing"
)

func TestAgentEnvReplacesMark(t *testing.T) {
	got := agentEnv([]string{"A=1", "OBK_AGENT=0"})
	if !slices.Equal(got, []string{"A=1", "OBK_AGENT=1"}) {
		t.Fatalf("env = %v", got)
	}
}

func TestBashToolMarksChildEnv(t *testing.T) {
	t.Setenv("OBK_AGENT", "0")
	out, err := NewBashTool(0).runCommand(context.Background(), "echo $OBK_AGENT")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if strings.TrimSpace(out) != "1" {
		t.Fatalf("OBK_AGENT = %q, want 1", out)
	}
}
