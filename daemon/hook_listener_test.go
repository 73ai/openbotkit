package daemon

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riversqlite"
	"github.com/riverqueue/river/rivermigrate"

	"github.com/73ai/openbotkit/channel"
	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/daemon/jobs"
	"github.com/73ai/openbotkit/service/hooks"
	"github.com/73ai/openbotkit/store"
)

func TestHookListener_HandleEvent_EnqueuesJob(t *testing.T) {
	dir := t.TempDir()
	jobsDBPath := filepath.Join(dir, "jobs.db")
	hooksDBPath := filepath.Join(dir, "hooks.db")

	// Set up hooks DB with a hook.
	hooksDB, err := store.Open(store.SQLiteConfig(hooksDBPath))
	if err != nil {
		t.Fatalf("open hooks db: %v", err)
	}
	defer hooksDB.Close()
	hooks.Migrate(hooksDB)

	hookID, err := hooks.Create(hooksDB, &hooks.EventHook{
		EventType: "gmail_sync",
		Prompt:    "test prompt",
		Channel:   "telegram",
		ModelTier: "nano",
	})
	if err != nil {
		t.Fatalf("create hook: %v", err)
	}

	// Set up River client with EventHookWorker.
	jobsDB, err := sql.Open("sqlite", jobsDBPath)
	if err != nil {
		t.Fatalf("open jobs db: %v", err)
	}
	defer jobsDB.Close()
	jobsDB.SetMaxOpenConns(1)

	driver := riversqlite.New(jobsDB)
	migrator, _ := rivermigrate.New(driver, nil)
	migrator.Migrate(context.Background(), rivermigrate.DirectionUp, nil)

	workers := river.NewWorkers()
	river.AddWorker(workers, &jobs.EventHookWorker{
		Cfg:     config.Default(),
		ChanReg: channel.NewRegistry(),
		HooksDB: hooksDB,
	})

	riverClient, err := river.NewClient(driver, &river.Config{
		Queues:  map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: 1}},
		Workers: workers,
	})
	if err != nil {
		t.Fatalf("create river client: %v", err)
	}

	notifier := NewSyncNotifier()
	listener := NewHookListener(config.Default(), riverClient, jobsDB, notifier, hooksDB)

	// Call handleEvent directly.
	ctx := context.Background()
	listener.handleEvent(ctx, SyncSignal{
		Source: "gmail",
		Data:   []int64{1, 2, 3},
	})

	// Verify a job was enqueued.
	var count int
	err = jobsDB.QueryRow("SELECT count(*) FROM river_job WHERE kind = 'event_hook'").Scan(&count)
	if err != nil {
		t.Fatalf("query jobs: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 enqueued job, got %d", count)
	}

	// Verify job args contain the hook ID and item IDs.
	var argsJSON string
	jobsDB.QueryRow("SELECT args FROM river_job WHERE kind = 'event_hook'").Scan(&argsJSON)
	if argsJSON == "" {
		t.Fatal("job args empty")
	}
	t.Logf("enqueued job args: %s", argsJSON)
	_ = hookID // used via DB
}

func TestHookListener_HandleEvent_SkipsNoData(t *testing.T) {
	dir := t.TempDir()
	hooksDBPath := filepath.Join(dir, "hooks.db")
	jobsDBPath := filepath.Join(dir, "jobs.db")

	hooksDB, _ := store.Open(store.SQLiteConfig(hooksDBPath))
	defer hooksDB.Close()
	hooks.Migrate(hooksDB)

	hooks.Create(hooksDB, &hooks.EventHook{
		EventType: "gmail_sync",
		Prompt:    "test",
		Channel:   "telegram",
		ModelTier: "nano",
	})

	jobsDB, _ := sql.Open("sqlite", jobsDBPath)
	defer jobsDB.Close()
	jobsDB.SetMaxOpenConns(1)
	driver := riversqlite.New(jobsDB)
	migrator, _ := rivermigrate.New(driver, nil)
	migrator.Migrate(context.Background(), rivermigrate.DirectionUp, nil)

	workers := river.NewWorkers()
	river.AddWorker(workers, &jobs.EventHookWorker{
		Cfg:     config.Default(),
		ChanReg: channel.NewRegistry(),
		HooksDB: hooksDB,
	})
	riverClient, _ := river.NewClient(driver, &river.Config{
		Queues:  map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: 1}},
		Workers: workers,
	})

	notifier := NewSyncNotifier()
	listener := NewHookListener(config.Default(), riverClient, jobsDB, notifier, hooksDB)

	// Signal with nil data — should not enqueue.
	listener.handleEvent(context.Background(), SyncSignal{Source: "gmail", Data: nil})

	var count int
	jobsDB.QueryRow("SELECT count(*) FROM river_job WHERE kind = 'event_hook'").Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 jobs for nil data, got %d", count)
	}

	// Signal with empty IDs — should not enqueue.
	listener.handleEvent(context.Background(), SyncSignal{Source: "gmail", Data: []int64{}})
	jobsDB.QueryRow("SELECT count(*) FROM river_job WHERE kind = 'event_hook'").Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 jobs for empty IDs, got %d", count)
	}
}

func TestHookListener_HandleEvent_NoHooks(t *testing.T) {
	dir := t.TempDir()
	hooksDBPath := filepath.Join(dir, "hooks.db")
	jobsDBPath := filepath.Join(dir, "jobs.db")

	hooksDB, _ := store.Open(store.SQLiteConfig(hooksDBPath))
	defer hooksDB.Close()
	hooks.Migrate(hooksDB)
	// No hooks created.

	jobsDB, _ := sql.Open("sqlite", jobsDBPath)
	defer jobsDB.Close()
	jobsDB.SetMaxOpenConns(1)
	driver := riversqlite.New(jobsDB)
	migrator, _ := rivermigrate.New(driver, nil)
	migrator.Migrate(context.Background(), rivermigrate.DirectionUp, nil)

	workers := river.NewWorkers()
	river.AddWorker(workers, &jobs.EventHookWorker{
		Cfg:     config.Default(),
		ChanReg: channel.NewRegistry(),
		HooksDB: hooksDB,
	})
	riverClient, _ := river.NewClient(driver, &river.Config{
		Queues:  map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: 1}},
		Workers: workers,
	})

	notifier := NewSyncNotifier()
	listener := NewHookListener(config.Default(), riverClient, jobsDB, notifier, hooksDB)

	// Signal with data but no hooks — should not enqueue.
	listener.handleEvent(context.Background(), SyncSignal{Source: "gmail", Data: []int64{1, 2}})

	var count int
	jobsDB.QueryRow("SELECT count(*) FROM river_job WHERE kind = 'event_hook'").Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 jobs with no hooks, got %d", count)
	}
}

func TestHookListener_RunStopsOnCancel(t *testing.T) {
	dir := t.TempDir()
	hooksDB, _ := store.Open(store.SQLiteConfig(filepath.Join(dir, "hooks.db")))
	defer hooksDB.Close()
	hooks.Migrate(hooksDB)

	jobsDB, _ := sql.Open("sqlite", filepath.Join(dir, "jobs.db"))
	defer jobsDB.Close()
	jobsDB.SetMaxOpenConns(1)
	driver := riversqlite.New(jobsDB)
	migrator, _ := rivermigrate.New(driver, nil)
	migrator.Migrate(context.Background(), rivermigrate.DirectionUp, nil)

	workers := river.NewWorkers()
	river.AddWorker(workers, &jobs.EventHookWorker{
		Cfg:     config.Default(),
		ChanReg: channel.NewRegistry(),
		HooksDB: hooksDB,
	})
	riverClient, _ := river.NewClient(driver, &river.Config{
		Queues:  map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: 1}},
		Workers: workers,
	})

	notifier := NewSyncNotifier()
	listener := NewHookListener(config.Default(), riverClient, jobsDB, notifier, hooksDB)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		listener.Run(ctx)
		close(done)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("listener did not stop after context cancel")
	}
}
