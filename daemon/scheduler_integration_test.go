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
	"github.com/robfig/cron/v3"

	"github.com/priyanshujain/openbotkit/config"
	"github.com/priyanshujain/openbotkit/daemon/jobs"
	"github.com/priyanshujain/openbotkit/source/scheduler"
	"github.com/priyanshujain/openbotkit/store"
)

func TestSchedulerOneShotIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := t.TempDir()
	schedDBPath := filepath.Join(dir, "sched.db")
	jobsDBPath := filepath.Join(dir, "jobs.db")

	cfg := &config.Config{
		Scheduler: &config.SchedulerConfig{
			Storage: config.StorageConfig{Driver: "sqlite", DSN: schedDBPath},
		},
	}

	// Create scheduler DB and insert a due one-shot.
	sdb, err := store.Open(store.Config{Driver: "sqlite", DSN: schedDBPath})
	if err != nil {
		t.Fatalf("open sched db: %v", err)
	}
	if err := scheduler.Migrate(sdb); err != nil {
		t.Fatalf("migrate sched: %v", err)
	}
	past := time.Now().UTC().Add(-1 * time.Minute)
	id, err := scheduler.Create(sdb, &scheduler.Schedule{
		Type:        scheduler.OneShot,
		ScheduledAt: &past,
		Task:        "test task",
		Channel:     "telegram",
		ChannelMeta: scheduler.ChannelMeta{BotToken: "tok", OwnerID: 1},
		Timezone:    "UTC",
		Description: "integration test",
	})
	if err != nil {
		t.Fatalf("create schedule: %v", err)
	}
	sdb.Close()

	// Set up River client.
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
	ctx := context.Background()
	if _, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	workers := river.NewWorkers()
	river.AddWorker(workers, &jobs.ScheduledTaskWorker{Cfg: cfg})

	riverClient, err := river.NewClient(driver, &river.Config{
		Queues:  map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: 1}},
		Workers: workers,
	})
	if err != nil {
		t.Fatalf("create river client: %v", err)
	}

	// Create scheduler (don't start cron loops, just test pollOneShot).
	s := &Scheduler{
		cfg:     cfg,
		river:   riverClient,
		jobsDB:  jobsDB,
		cron:    cron.New(cron.WithLocation(time.UTC)),
		entries: make(map[int64]cron.EntryID),
	}

	if err := s.pollOneShot(ctx); err != nil {
		t.Fatalf("pollOneShot: %v", err)
	}

	// Verify the schedule was disabled (worker marks completed later).
	sdb, _ = store.Open(store.Config{Driver: "sqlite", DSN: schedDBPath})
	defer sdb.Close()
	got, err := scheduler.Get(sdb, id)
	if err != nil {
		t.Fatalf("get schedule: %v", err)
	}
	if got.Enabled {
		t.Fatal("expected schedule to be disabled after enqueue")
	}
}
