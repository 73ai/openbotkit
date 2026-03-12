package jobs

import (
	"encoding/json"
	"testing"
)

func TestScheduledTaskArgsKind(t *testing.T) {
	args := ScheduledTaskArgs{}
	if got := args.Kind(); got != "scheduled_task" {
		t.Errorf("Kind(): got %q, want %q", got, "scheduled_task")
	}
}

func TestScheduledTaskArgsSerialization(t *testing.T) {
	orig := ScheduledTaskArgs{
		ScheduleID:  42,
		Task:        "check weather",
		Channel:     "telegram",
		ChannelMeta: `{"bot_token":"tok","owner_id":123}`,
	}
	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got ScheduledTaskArgs
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.ScheduleID != orig.ScheduleID {
		t.Errorf("ScheduleID: got %d, want %d", got.ScheduleID, orig.ScheduleID)
	}
	if got.Task != orig.Task {
		t.Errorf("Task: got %q, want %q", got.Task, orig.Task)
	}
	if got.Channel != orig.Channel {
		t.Errorf("Channel: got %q, want %q", got.Channel, orig.Channel)
	}
}
