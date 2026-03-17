package spectest

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/73ai/openbotkit/agent"
	"github.com/73ai/openbotkit/agent/tools"
	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/source/scheduler"
	"github.com/73ai/openbotkit/store"
)

// agentWithScheduleTools creates an agent with schedule tools registered.
func agentWithScheduleTools(t *testing.T, fx *LocalFixture) (*agent.Agent, string) {
	t.Helper()

	schedDBPath := filepath.Join(fx.dir, "scheduler", "data.db")

	cfg := &config.Config{
		Scheduler: &config.SchedulerConfig{
			Storage: config.StorageConfig{Driver: "sqlite", DSN: schedDBPath},
		},
	}

	deps := tools.ScheduleToolDeps{
		Cfg:     cfg,
		Channel: "telegram",
		ChannelMeta: scheduler.ChannelMeta{
			BotToken: "test-token",
			OwnerID:  42,
		},
	}

	toolReg := tools.NewRegistry()
	toolReg.Register(tools.NewBashTool(30 * time.Second))
	toolReg.Register(&tools.FileReadTool{})
	toolReg.Register(&tools.LoadSkillsTool{})
	toolReg.Register(&tools.SearchSkillsTool{})
	toolReg.Register(tools.NewCreateScheduleTool(deps))
	toolReg.Register(tools.NewListSchedulesTool(deps))
	toolReg.Register(tools.NewDeleteScheduleTool(deps))

	identity := "You are a personal AI assistant powered by OpenBotKit.\n"
	extras := "\nThe user's timezone is America/New_York.\n"
	blocks := tools.BuildSystemBlocks(identity, toolReg, extras)

	a := agent.New(fx.Provider, fx.Model, toolReg,
		agent.WithSystemBlocks(blocks),
		agent.WithMaxIterations(10),
	)
	return a, schedDBPath
}

func TestSpec_CreateScheduleRecurring(t *testing.T) {
	EachProvider(t, func(t *testing.T, fx *LocalFixture) {
		a, schedDBPath := agentWithScheduleTools(t, fx)

		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		prompt := "Remind me to check the weather every day at 9am"
		result, err := a.Run(ctx, prompt)
		if err != nil {
			t.Fatalf("agent.Run: %v", err)
		}

		AssertNotEmpty(t, result)
		fx.AssertJudge(t, prompt, result,
			"The response should confirm that a daily schedule or reminder was created.")

		// Verify schedule exists in DB.
		db, err := store.Open(store.SQLiteConfig(schedDBPath))
		if err != nil {
			t.Fatalf("open sched db: %v", err)
		}
		defer db.Close()

		schedules, err := scheduler.ListEnabled(db)
		if err != nil {
			t.Fatalf("list: %v", err)
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
	})
}

func TestSpec_CreateAndListSchedules(t *testing.T) {
	EachProvider(t, func(t *testing.T, fx *LocalFixture) {
		a, _ := agentWithScheduleTools(t, fx)

		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		// First create a schedule.
		_, err := a.Run(ctx, "Schedule a task to check USD-EUR exchange rate every day at 10am")
		if err != nil {
			t.Fatalf("create: %v", err)
		}

		// Now list schedules.
		result, err := a.Run(ctx, "What tasks are scheduled?")
		if err != nil {
			t.Fatalf("list: %v", err)
		}

		AssertNotEmpty(t, result)
		fx.AssertJudge(t, "What tasks are scheduled?", result,
			"The response should list at least one scheduled task. "+
				"It should mention the exchange rate task or similar description.")
	})
}

func TestSpec_CreateAndDeleteSchedule(t *testing.T) {
	EachProvider(t, func(t *testing.T, fx *LocalFixture) {
		a, schedDBPath := agentWithScheduleTools(t, fx)

		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		// Create a schedule.
		_, err := a.Run(ctx, "Remind me to check the news every day at 8am")
		if err != nil {
			t.Fatalf("create: %v", err)
		}

		// Delete it.
		result, err := a.Run(ctx, "Delete schedule 1")
		if err != nil {
			t.Fatalf("delete: %v", err)
		}

		AssertNotEmpty(t, result)
		fx.AssertJudge(t, "Delete schedule 1", result,
			"The response should confirm the schedule was deleted.")

		// Verify DB is empty.
		db, err := store.Open(store.SQLiteConfig(schedDBPath))
		if err != nil {
			t.Fatalf("open db: %v", err)
		}
		defer db.Close()

		schedules, _ := scheduler.List(db)
		if len(schedules) != 0 {
			t.Fatalf("expected 0 schedules after delete, got %d", len(schedules))
		}
	})
}
