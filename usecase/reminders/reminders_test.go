package reminders

import (
	"context"
	"testing"
	"time"

	"github.com/73ai/openbotkit/service/scheduler"
	"github.com/73ai/openbotkit/spectest"
	"github.com/73ai/openbotkit/store"
	"github.com/73ai/openbotkit/usecase"
)

func TestUseCase_OneShotReminder(t *testing.T) {
	fx := usecase.NewFixture(t)
	a := fx.ScheduleAgent(t)

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	// Turn 1: ask for a one-shot reminder
	result, err := a.Run(ctx, "Remind me to call the dentist tomorrow at 3pm")
	if err != nil {
		t.Fatalf("turn 1 (create): %v", err)
	}
	spectest.AssertNotEmpty(t, result)

	// Turn 2: ask what reminders exist
	result, err = a.Run(ctx, "What reminders do I have?")
	if err != nil {
		t.Fatalf("turn 2 (list): %v", err)
	}
	spectest.AssertNotEmpty(t, result)
	fx.AssertJudge(t, "What reminders do I have?", result,
		"The response should list a reminder about calling the dentist.")

	// Verify DB state
	db, err := store.Open(store.SQLiteConfig(fx.SchedDBPath()))
	if err != nil {
		t.Fatalf("open sched db: %v", err)
	}
	defer db.Close()

	schedules, err := scheduler.List(db)
	if err != nil {
		t.Fatalf("list schedules: %v", err)
	}
	if len(schedules) == 0 {
		t.Fatal("expected at least one schedule in DB")
	}

	s := schedules[0]
	if s.Type != scheduler.OneShot {
		t.Errorf("expected one_shot, got %s", s.Type)
	}
	if s.ScheduledAt == nil {
		t.Error("expected scheduled_at to be set")
	}
}

func TestUseCase_RecurringReminderExecution(t *testing.T) {
	fx := usecase.NewFixture(t)
	a := fx.ScheduleAgent(t)

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	// Turn 1: ask for a daily recurring schedule
	result, err := a.Run(ctx,
		"Tell me the EUR/USD exchange rate on telegram every morning at 10am")
	if err != nil {
		t.Fatalf("turn 1 (create): %v", err)
	}
	spectest.AssertNotEmpty(t, result)

	// Verify schedule was created correctly in DB
	db, err := store.Open(store.SQLiteConfig(fx.SchedDBPath()))
	if err != nil {
		t.Fatalf("open sched db: %v", err)
	}
	defer db.Close()

	schedules, err := scheduler.ListEnabled(db)
	if err != nil {
		t.Fatalf("list schedules: %v", err)
	}
	if len(schedules) == 0 {
		t.Fatal("expected at least one schedule in DB")
	}

	s := schedules[0]
	if s.Type != scheduler.Recurring {
		t.Errorf("expected recurring, got %s", s.Type)
	}
	if s.CronExpr == "" {
		t.Error("expected cron expression to be set")
	}
	if s.Task == "" {
		t.Fatal("expected non-empty task prompt in schedule")
	}
	t.Logf("stored task prompt: %s", s.Task)
	t.Logf("cron expression: %s", s.CronExpr)

	// Simulate execution: run the stored task through a fresh agent
	// with web tools, same as what the daemon worker does in production.
	taskAgent := fx.Agent(t)
	taskResult, err := taskAgent.Run(ctx, s.Task)
	if err != nil {
		t.Fatalf("task execution: %v", err)
	}

	spectest.AssertNotEmpty(t, taskResult)
	fx.AssertJudge(t, s.Task, taskResult,
		"The response should mention EUR/USD or euro-dollar and include an exchange rate number.")
}
