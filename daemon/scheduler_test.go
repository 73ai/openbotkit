package daemon

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/source/scheduler"
	"github.com/73ai/openbotkit/store"
)

func TestIsValidFrequency(t *testing.T) {
	s := &Scheduler{}

	tests := []struct {
		expr  string
		valid bool
	}{
		{"0 9 * * *", true},    // daily at 9am
		{"0 */2 * * *", true},  // every 2 hours
		{"*/5 * * * *", false}, // every 5 minutes
		{"0 * * * *", true},    // every hour
	}

	for _, tt := range tests {
		got := s.isValidFrequency(tt.expr)
		if got != tt.valid {
			t.Errorf("isValidFrequency(%q): got %v, want %v", tt.expr, got, tt.valid)
		}
	}
}

func TestLoadSchedulesAddRemove(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "sched.db")

	cfg := &config.Config{
		Scheduler: &config.SchedulerConfig{
			Storage: config.StorageConfig{Driver: "sqlite", DSN: dbPath},
		},
	}

	db, err := store.Open(store.Config{Driver: "sqlite", DSN: dbPath})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := scheduler.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	id, err := scheduler.Create(db, &scheduler.Schedule{
		Type:        scheduler.Recurring,
		CronExpr:    "0 9 * * *",
		Task:        "check weather",
		Channel:     "telegram",
		ChannelMeta: scheduler.ChannelMeta{BotToken: "tok", OwnerID: 1},
		Timezone:    "UTC",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	db.Close()

	s := &Scheduler{
		cfg:     cfg,
		entries: make(map[int64]cron.EntryID),
	}

	s.cron = cron.New(cron.WithLocation(time.UTC))
	s.cron.Start()
	defer s.cron.Stop()

	if err := s.loadSchedules(); err != nil {
		t.Fatalf("load: %v", err)
	}
	if _, ok := s.entries[id]; !ok {
		t.Fatal("expected cron entry for schedule")
	}

	db, _ = store.Open(store.Config{Driver: "sqlite", DSN: dbPath})
	scheduler.Delete(db, id)
	db.Close()

	if err := s.loadSchedules(); err != nil {
		t.Fatalf("reload: %v", err)
	}
	if _, ok := s.entries[id]; ok {
		t.Fatal("expected cron entry removed after delete")
	}
}
