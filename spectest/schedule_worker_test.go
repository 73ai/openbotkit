package spectest

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riversqlite"
	"github.com/riverqueue/river/rivermigrate"

	"github.com/73ai/openbotkit/agent"
	"github.com/73ai/openbotkit/agent/tools"
	"github.com/73ai/openbotkit/channel"
	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/daemon/jobs"
	"github.com/73ai/openbotkit/provider"
	"github.com/73ai/openbotkit/service/scheduler"
	"github.com/73ai/openbotkit/store"
)

type capturePusher struct {
	mu       sync.Mutex
	messages []string
}

func (p *capturePusher) Push(_ context.Context, message string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.messages = append(p.messages, message)
	return nil
}

func (p *capturePusher) Messages() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	cp := make([]string, len(p.messages))
	copy(cp, p.messages)
	return cp
}

// makeAgentRunner creates an AgentRunner that uses the given LLM provider.
func makeAgentRunner(p provider.Provider, model string) jobs.AgentRunner {
	return func(ctx context.Context, task string) (string, error) {
		toolReg := tools.NewStandardRegistry(nil, nil)
		identity := "You are a scheduled task agent. Execute the task and return a concise result.\n"
		blocks := tools.BuildSystemBlocks(identity, toolReg)
		a := agent.New(p, model, toolReg, agent.WithSystemBlocks(blocks))
		return a.Run(ctx, task)
	}
}

// TestSpec_ScheduledTaskWorkerExecution runs a real scheduled task through the
// River worker with a real LLM, capturing the pushed output via a mock pusher.
func TestSpec_ScheduledTaskWorkerExecution(t *testing.T) {
	fx := NewLocalFixture(t)

	dir := fx.dir
	schedDBPath := filepath.Join(dir, "scheduler", "data.db")
	jobsDBPath := filepath.Join(dir, "jobs.db")

	// Create scheduler DB and insert a one-shot schedule.
	sdb, err := store.Open(store.SQLiteConfig(schedDBPath))
	if err != nil {
		t.Fatalf("open sched db: %v", err)
	}
	if err := scheduler.Migrate(sdb); err != nil {
		t.Fatalf("migrate sched: %v", err)
	}

	meta := scheduler.ChannelMeta{BotToken: "test", OwnerID: 1}
	metaJSON, _ := json.Marshal(meta)

	past := time.Now().UTC().Add(-1 * time.Minute)
	scheduleID, err := scheduler.Create(sdb, &scheduler.Schedule{
		Type:        scheduler.OneShot,
		ScheduledAt: &past,
		Task:        "What is 2 + 2? Reply with just the number.",
		Channel:     "test",
		ChannelMeta: meta,
		Timezone:    "UTC",
		Description: "math test",
	})
	if err != nil {
		t.Fatalf("create schedule: %v", err)
	}
	sdb.Close()

	cfg := &config.Config{
		Scheduler: &config.SchedulerConfig{
			Storage: config.StorageConfig{Driver: "sqlite", DSN: schedDBPath},
		},
	}

	// Set up River.
	jobsDB, err := sql.Open("sqlite", jobsDBPath)
	if err != nil {
		t.Fatalf("open jobs db: %v", err)
	}
	defer jobsDB.Close()
	jobsDB.SetMaxOpenConns(1)

	driver := riversqlite.New(jobsDB)
	migrator, err := rivermigrate.New(driver, nil)
	if err != nil {
		t.Fatalf("create migrator: %v", err)
	}
	if _, err := migrator.Migrate(context.Background(), rivermigrate.DirectionUp, nil); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	// Create worker with injected agent runner and mock pusher.
	pusher := &capturePusher{}
	worker := &jobs.ScheduledTaskWorker{
		Cfg:          cfg,
		RunAgentFunc: makeAgentRunner(fx.Provider, fx.Model),
		MakePusher: func(_ string, _ scheduler.ChannelMeta) (channel.Pusher, error) {
			return pusher, nil
		},
	}

	workers := river.NewWorkers()
	river.AddWorker(workers, worker)

	riverClient, err := river.NewClient(driver, &river.Config{
		Queues:  map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: 1}},
		Workers: workers,
	})
	if err != nil {
		t.Fatalf("create river client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	if err := riverClient.Start(ctx); err != nil {
		t.Fatalf("start river: %v", err)
	}
	defer riverClient.Stop(context.Background())

	// Insert the job.
	tx, err := jobsDB.Begin()
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	_, err = riverClient.InsertTx(ctx, tx, jobs.ScheduledTaskArgs{
		ScheduleID:  scheduleID,
		Task:        "What is 2 + 2? Reply with just the number.",
		Channel:     "test",
		ChannelMeta: string(metaJSON),
	}, &river.InsertOpts{MaxAttempts: 2})
	if err != nil {
		tx.Rollback()
		t.Fatalf("insert job: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	// Wait for the pusher to receive a message.
	deadline := time.After(90 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for scheduled task to deliver result")
		case <-ticker.C:
			msgs := pusher.Messages()
			if len(msgs) > 0 {
				t.Logf("pushed message: %s", msgs[0])

				// Verify schedule was updated.
				sdb, _ := store.Open(store.SQLiteConfig(schedDBPath))
				defer sdb.Close()
				s, err := scheduler.Get(sdb, scheduleID)
				if err != nil {
					t.Fatalf("get schedule: %v", err)
				}
				if s.LastRunAt == nil {
					t.Error("expected last_run_at to be set")
				}
				if s.LastError != "" {
					t.Errorf("expected no last_error, got %q", s.LastError)
				}
				if s.CompletedAt == nil {
					t.Error("expected one-shot to be marked completed")
				}
				return
			}
		}
	}
}
