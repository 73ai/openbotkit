package daemon

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riversqlite"
	"github.com/riverqueue/river/rivermigrate"
	"github.com/robfig/cron/v3"

	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/daemon/jobs"
	"github.com/73ai/openbotkit/service/scheduler"
	"github.com/73ai/openbotkit/store"
)

type testSchedulerEnv struct {
	cfg         *config.Config
	schedDBPath string
	jobsDB      *sql.DB
	riverClient *river.Client[*sql.Tx]
	scheduler   *Scheduler
	ctx         context.Context
}

func setupTestScheduler(t *testing.T) *testSchedulerEnv {
	t.Helper()
	dir := t.TempDir()
	schedDBPath := filepath.Join(dir, "sched.db")
	jobsDBPath := filepath.Join(dir, "jobs.db")

	cfg := &config.Config{
		Scheduler: &config.SchedulerConfig{
			Storage: config.StorageConfig{Driver: "sqlite", DSN: schedDBPath},
		},
	}

	jobsDB, err := sql.Open("sqlite", jobsDBPath)
	if err != nil {
		t.Fatalf("open jobs db: %v", err)
	}
	t.Cleanup(func() { jobsDB.Close() })
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
	river.AddWorker(workers, &jobs.BackupWorker{Cfg: cfg})

	riverClient, err := river.NewClient(driver, &river.Config{
		Queues:  map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: 1}},
		Workers: workers,
	})
	if err != nil {
		t.Fatalf("create river client: %v", err)
	}

	s := &Scheduler{
		cfg:     cfg,
		river:   riverClient,
		jobsDB:  jobsDB,
		ctx:     ctx,
		cron:    cron.New(cron.WithLocation(time.UTC)),
		entries: make(map[int64]cron.EntryID),
	}

	return &testSchedulerEnv{
		cfg:         cfg,
		schedDBPath: schedDBPath,
		jobsDB:      jobsDB,
		riverClient: riverClient,
		scheduler:   s,
		ctx:         ctx,
	}
}

func (e *testSchedulerEnv) openSchedDB(t *testing.T) *store.DB {
	t.Helper()
	sdb, err := store.Open(store.Config{Driver: "sqlite", DSN: e.schedDBPath})
	if err != nil {
		t.Fatalf("open sched db: %v", err)
	}
	return sdb
}

func (e *testSchedulerEnv) migrateSchedDB(t *testing.T) *store.DB {
	t.Helper()
	sdb := e.openSchedDB(t)
	if err := scheduler.Migrate(sdb); err != nil {
		sdb.Close()
		t.Fatalf("migrate sched: %v", err)
	}
	return sdb
}

func (e *testSchedulerEnv) countJobs(t *testing.T, kind string) int {
	t.Helper()
	var count int
	err := e.jobsDB.QueryRow("SELECT count(*) FROM river_job WHERE kind = ?", kind).Scan(&count)
	if err != nil {
		t.Fatalf("query river_job: %v", err)
	}
	return count
}

func setTestConfigDir(t *testing.T) string {
	t.Helper()
	obkDir := filepath.Join(t.TempDir(), ".obk")
	os.MkdirAll(obkDir, 0o755)
	t.Setenv("OBK_CONFIG_DIR", obkDir)
	return obkDir
}

func TestCheckOverdueRecurringEnqueuesJob(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestScheduler(t)
	sdb := env.migrateSchedDB(t)

	id, err := scheduler.Create(sdb, &scheduler.Schedule{
		Type:        scheduler.Recurring,
		CronExpr:    "0 */2 * * *",
		Task:        "overdue task",
		Channel:     "telegram",
		ChannelMeta: scheduler.ChannelMeta{BotToken: "tok", OwnerID: 1},
		Timezone:    "UTC",
	})
	if err != nil {
		t.Fatalf("create schedule: %v", err)
	}

	threeHoursAgo := time.Now().UTC().Add(-3 * time.Hour)
	if err := scheduler.UpdateLastRun(sdb, id, threeHoursAgo, ""); err != nil {
		t.Fatalf("update last run: %v", err)
	}
	sdb.Close()

	env.scheduler.checkOverdueRecurring()

	if got := env.countJobs(t, "scheduled_task"); got != 1 {
		t.Errorf("expected 1 overdue job, got %d", got)
	}

	sdb = env.openSchedDB(t)
	defer sdb.Close()
	fresh, err := scheduler.Get(sdb, id)
	if err != nil {
		t.Fatalf("get schedule: %v", err)
	}
	if fresh.LastRunAt == nil || time.Since(*fresh.LastRunAt) > time.Minute {
		t.Errorf("expected LastRunAt to be stamped recently, got %v", fresh.LastRunAt)
	}
}

func TestCheckOverdueRecurringSkipsRecentRun(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestScheduler(t)
	sdb := env.migrateSchedDB(t)

	id, err := scheduler.Create(sdb, &scheduler.Schedule{
		Type:        scheduler.Recurring,
		CronExpr:    "0 */6 * * *",
		Task:        "recent task",
		Channel:     "telegram",
		ChannelMeta: scheduler.ChannelMeta{BotToken: "tok", OwnerID: 1},
		Timezone:    "UTC",
	})
	if err != nil {
		t.Fatalf("create schedule: %v", err)
	}

	recent := time.Now().UTC().Add(-30 * time.Minute)
	if err := scheduler.UpdateLastRun(sdb, id, recent, ""); err != nil {
		t.Fatalf("update last run: %v", err)
	}
	sdb.Close()

	env.scheduler.checkOverdueRecurring()

	if got := env.countJobs(t, "scheduled_task"); got != 0 {
		t.Errorf("expected 0 jobs for recent schedule, got %d", got)
	}
}

func TestCheckOverdueRecurringNoDuplicate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestScheduler(t)
	sdb := env.migrateSchedDB(t)

	id, err := scheduler.Create(sdb, &scheduler.Schedule{
		Type:        scheduler.Recurring,
		CronExpr:    "0 */2 * * *",
		Task:        "dedup task",
		Channel:     "telegram",
		ChannelMeta: scheduler.ChannelMeta{BotToken: "tok", OwnerID: 1},
		Timezone:    "UTC",
	})
	if err != nil {
		t.Fatalf("create schedule: %v", err)
	}

	threeHoursAgo := time.Now().UTC().Add(-3 * time.Hour)
	if err := scheduler.UpdateLastRun(sdb, id, threeHoursAgo, ""); err != nil {
		t.Fatalf("update last run: %v", err)
	}
	sdb.Close()

	env.scheduler.checkOverdueRecurring()
	env.scheduler.checkOverdueRecurring()

	if got := env.countJobs(t, "scheduled_task"); got != 1 {
		t.Errorf("expected 1 job after two runs (dedup), got %d", got)
	}
}

func TestCheckOverdueBackupEnqueues(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	obkDir := setTestConfigDir(t)
	env := setupTestScheduler(t)

	if err := config.LinkSource("backup"); err != nil {
		t.Fatalf("link backup source: %v", err)
	}

	backupDir := filepath.Join(obkDir, "backup")
	manifestPath := filepath.Join(backupDir, "last_manifest.json")
	staleTime := time.Now().Add(-25 * time.Hour)
	manifest := map[string]any{
		"version":   1,
		"id":        "old-backup",
		"timestamp": staleTime.Format(time.RFC3339),
		"hostname":  "test",
		"files":     map[string]any{},
	}
	data, _ := json.Marshal(manifest)
	os.WriteFile(manifestPath, data, 0o644)

	env.cfg.Backup = &config.BackupConfig{
		Enabled:  true,
		Schedule: "24h",
	}

	env.scheduler.checkOverdueBackup()

	if got := env.countJobs(t, "backup"); got != 1 {
		t.Errorf("expected 1 backup job, got %d", got)
	}
}

func TestCheckOverdueBackupSkipsRecent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	obkDir := setTestConfigDir(t)
	env := setupTestScheduler(t)

	if err := config.LinkSource("backup"); err != nil {
		t.Fatalf("link backup source: %v", err)
	}

	backupDir := filepath.Join(obkDir, "backup")
	manifestPath := filepath.Join(backupDir, "last_manifest.json")
	recentTime := time.Now().Add(-1 * time.Hour)
	manifest := map[string]any{
		"version":   1,
		"id":        "recent-backup",
		"timestamp": recentTime.Format(time.RFC3339),
		"hostname":  "test",
		"files":     map[string]any{},
	}
	data, _ := json.Marshal(manifest)
	os.WriteFile(manifestPath, data, 0o644)

	env.cfg.Backup = &config.BackupConfig{
		Enabled:  true,
		Schedule: "24h",
	}

	env.scheduler.checkOverdueBackup()

	if got := env.countJobs(t, "backup"); got != 0 {
		t.Errorf("expected 0 backup jobs for recent manifest, got %d", got)
	}
}

func TestTickRunsAllChecks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	obkDir := setTestConfigDir(t)
	env := setupTestScheduler(t)
	sdb := env.migrateSchedDB(t)

	_, err := scheduler.Create(sdb, &scheduler.Schedule{
		Type:        scheduler.Recurring,
		CronExpr:    "0 */2 * * *",
		Task:        "recurring task",
		Channel:     "telegram",
		ChannelMeta: scheduler.ChannelMeta{BotToken: "tok", OwnerID: 1},
		Timezone:    "UTC",
	})
	if err != nil {
		t.Fatalf("create recurring: %v", err)
	}
	threeHoursAgo := time.Now().UTC().Add(-3 * time.Hour)
	if err := scheduler.UpdateLastRun(sdb, 1, threeHoursAgo, ""); err != nil {
		t.Fatalf("update last run: %v", err)
	}

	past := time.Now().UTC().Add(-1 * time.Minute)
	_, err = scheduler.Create(sdb, &scheduler.Schedule{
		Type:        scheduler.OneShot,
		ScheduledAt: &past,
		Task:        "oneshot task",
		Channel:     "telegram",
		ChannelMeta: scheduler.ChannelMeta{BotToken: "tok", OwnerID: 1},
		Timezone:    "UTC",
	})
	if err != nil {
		t.Fatalf("create one-shot: %v", err)
	}
	sdb.Close()

	if err := config.LinkSource("backup"); err != nil {
		t.Fatalf("link backup source: %v", err)
	}
	backupDir := filepath.Join(obkDir, "backup")
	manifestPath := filepath.Join(backupDir, "last_manifest.json")
	staleTime := time.Now().Add(-25 * time.Hour)
	manifest := map[string]any{
		"version":   1,
		"id":        "old-backup",
		"timestamp": staleTime.Format(time.RFC3339),
		"hostname":  "test",
		"files":     map[string]any{},
	}
	data, _ := json.Marshal(manifest)
	os.WriteFile(manifestPath, data, 0o644)

	env.cfg.Backup = &config.BackupConfig{
		Enabled:  true,
		Schedule: "24h",
	}

	env.scheduler.tick(env.ctx)

	if got := env.countJobs(t, "scheduled_task"); got != 2 {
		t.Errorf("expected 2 scheduled_task jobs (1 recurring + 1 one-shot), got %d", got)
	}
	if got := env.countJobs(t, "backup"); got != 1 {
		t.Errorf("expected 1 backup job, got %d", got)
	}
}

func TestSchedulerOneShotIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestScheduler(t)
	sdb := env.migrateSchedDB(t)

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

	if err := env.scheduler.pollOneShot(env.ctx); err != nil {
		t.Fatalf("pollOneShot: %v", err)
	}

	sdb = env.openSchedDB(t)
	defer sdb.Close()
	got, err := scheduler.Get(sdb, id)
	if err != nil {
		t.Fatalf("get schedule: %v", err)
	}
	if got.Enabled {
		t.Fatal("expected schedule to be disabled after enqueue")
	}
}
