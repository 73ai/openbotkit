package tools

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/priyanshujain/openbotkit/config"
	"github.com/priyanshujain/openbotkit/source/scheduler"
)

func testScheduleDeps(t *testing.T) ScheduleToolDeps {
	t.Helper()
	dir := t.TempDir()
	return ScheduleToolDeps{
		Cfg: &config.Config{
			Scheduler: &config.SchedulerConfig{
				Storage: config.StorageConfig{
					Driver: "sqlite",
					DSN:    filepath.Join(dir, "sched.db"),
				},
			},
		},
		Channel:     "telegram",
		ChannelMeta: scheduler.ChannelMeta{BotToken: "tok", OwnerID: 42},
	}
}

func TestCreateScheduleRecurring(t *testing.T) {
	deps := testScheduleDeps(t)
	tool := NewCreateScheduleTool(deps)

	input := `{
		"type": "recurring",
		"cron_expr": "0 9 * * *",
		"task": "Check weather forecast and report",
		"timezone": "UTC",
		"description": "Daily weather"
	}`
	result, err := tool.Execute(context.Background(), json.RawMessage(input))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(result, "Schedule created") {
		t.Errorf("unexpected result: %s", result)
	}
}

func TestCreateScheduleOneShot(t *testing.T) {
	deps := testScheduleDeps(t)
	tool := NewCreateScheduleTool(deps)

	future := time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339)
	input := `{
		"type": "one_shot",
		"scheduled_at": "` + future + `",
		"task": "Look up AAPL stock price",
		"timezone": "America/New_York",
		"description": "Stock check"
	}`
	result, err := tool.Execute(context.Background(), json.RawMessage(input))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(result, "Schedule created") {
		t.Errorf("unexpected result: %s", result)
	}
}

func TestCreateScheduleInvalidTimezone(t *testing.T) {
	deps := testScheduleDeps(t)
	tool := NewCreateScheduleTool(deps)

	input := `{
		"type": "recurring",
		"cron_expr": "0 9 * * *",
		"task": "do something",
		"timezone": "Invalid/Zone"
	}`
	_, err := tool.Execute(context.Background(), json.RawMessage(input))
	if err == nil {
		t.Fatal("expected error for invalid timezone")
	}
}

func TestCreateScheduleTooFrequent(t *testing.T) {
	deps := testScheduleDeps(t)
	tool := NewCreateScheduleTool(deps)

	input := `{
		"type": "recurring",
		"cron_expr": "*/5 * * * *",
		"task": "do something",
		"timezone": "UTC"
	}`
	_, err := tool.Execute(context.Background(), json.RawMessage(input))
	if err == nil {
		t.Fatal("expected error for too-frequent schedule")
	}
	if !strings.Contains(err.Error(), "once per hour") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCreateScheduleOneShotPast(t *testing.T) {
	deps := testScheduleDeps(t)
	tool := NewCreateScheduleTool(deps)

	past := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)
	input := `{
		"type": "one_shot",
		"scheduled_at": "` + past + `",
		"task": "do something",
		"timezone": "UTC"
	}`
	_, err := tool.Execute(context.Background(), json.RawMessage(input))
	if err == nil {
		t.Fatal("expected error for past scheduled_at")
	}
}

func TestListAndDeleteSchedules(t *testing.T) {
	deps := testScheduleDeps(t)

	createTool := NewCreateScheduleTool(deps)
	input := `{
		"type": "recurring",
		"cron_expr": "0 9 * * *",
		"task": "check weather",
		"timezone": "UTC",
		"description": "weather check"
	}`
	result, err := createTool.Execute(context.Background(), json.RawMessage(input))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if !strings.Contains(result, "ID: 1") {
		t.Fatalf("expected ID 1, got: %s", result)
	}

	listTool := NewListSchedulesTool(deps)
	result, err = listTool.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(result, "weather check") {
		t.Errorf("list should contain description: %s", result)
	}

	deleteTool := NewDeleteScheduleTool(deps)
	result, err = deleteTool.Execute(context.Background(), json.RawMessage(`{"id": 1}`))
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if !strings.Contains(result, "deleted") {
		t.Errorf("unexpected result: %s", result)
	}

	result, _ = listTool.Execute(context.Background(), json.RawMessage(`{}`))
	if !strings.Contains(result, "No scheduled tasks") {
		t.Errorf("expected empty list: %s", result)
	}
}
