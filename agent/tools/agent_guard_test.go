package tools

import (
	"context"
	"slices"
	"strings"
	"testing"
)

func TestAgentRunnersMarkChildEnv(t *testing.T) {
	t.Setenv("OBK_AGENT", "0")
	for _, kind := range []AgentKind{AgentClaude, AgentGemini, AgentCodex} {
		info := AgentInfo{Kind: kind, Binary: "x"}
		for name, env := range map[string][]string{
			"runner": NewAgentRunner(info).buildEnv(),
			"stream": NewStreamRunner(info).buildEnv(),
		} {
			var marks []string
			for _, e := range env {
				if strings.HasPrefix(e, "OBK_AGENT=") {
					marks = append(marks, e)
				}
			}
			if !slices.Equal(marks, []string{"OBK_AGENT=1"}) {
				t.Errorf("%s %v: OBK_AGENT entries = %v", name, kind, marks)
			}
		}
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
