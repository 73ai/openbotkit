package tools

import (
	"strings"
	"testing"
)

func TestBuildBaseSystemPrompt_IncludesScheduledTasks(t *testing.T) {
	reg := NewRegistry()
	reg.Register(NewCreateScheduleTool(ScheduleToolDeps{}))

	prompt := BuildBaseSystemPrompt(reg)
	if !strings.Contains(prompt, "Scheduled Tasks") {
		t.Error("expected 'Scheduled Tasks' section in prompt")
	}
	if !strings.Contains(prompt, "create_schedule") {
		t.Error("expected 'create_schedule' mention in prompt")
	}
}

func TestBuildBaseSystemPrompt_OmitsScheduledTasksWhenNotRegistered(t *testing.T) {
	reg := NewRegistry()

	prompt := BuildBaseSystemPrompt(reg)
	if strings.Contains(prompt, "Scheduled Tasks") {
		t.Error("prompt should not include 'Scheduled Tasks' without schedule tools")
	}
}
