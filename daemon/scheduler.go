package daemon

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/riverqueue/river"
	"github.com/robfig/cron/v3"

	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/daemon/jobs"
	backupsvc "github.com/73ai/openbotkit/service/backup"
	"github.com/73ai/openbotkit/service/scheduler"
	"github.com/73ai/openbotkit/store"
)

type Scheduler struct {
	cfg      *config.Config
	river    *river.Client[*sql.Tx]
	jobsDB   *sql.DB
	cron     *cron.Cron
	mu       sync.Mutex
	entries  map[int64]cron.EntryID
	ctx      context.Context
	notifier *SyncNotifier
}

func NewScheduler(cfg *config.Config, riverClient *river.Client[*sql.Tx], jobsDB *sql.DB, notifier *SyncNotifier) *Scheduler {
	return &Scheduler{
		cfg:      cfg,
		river:    riverClient,
		jobsDB:   jobsDB,
		entries:  make(map[int64]cron.EntryID),
		notifier: notifier,
	}
}

func (s *Scheduler) Start(ctx context.Context) error {
	db, err := s.openDB()
	if err != nil {
		return fmt.Errorf("open scheduler db: %w", err)
	}
	if err := scheduler.Migrate(db); err != nil {
		db.Close()
		return fmt.Errorf("migrate scheduler db: %w", err)
	}
	db.Close()

	s.ctx = ctx
	s.cron = cron.New(cron.WithLocation(time.UTC))
	s.cron.Start()

	s.tick(ctx)

	go s.tickLoop(ctx)
	if s.notifier != nil {
		go s.reactiveCheckLoop(ctx)
	}

	slog.Info("scheduler started")
	return nil
}

func (s *Scheduler) Stop() {
	if s.cron != nil {
		s.cron.Stop()
	}
	slog.Info("scheduler stopped")
}

func (s *Scheduler) tickLoop(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *Scheduler) tick(ctx context.Context) {
	if err := s.loadSchedules(); err != nil {
		slog.Error("scheduler: reload failed", "error", err)
	}
	s.checkOverdueRecurring()
	if err := s.pollOneShot(ctx); err != nil {
		slog.Error("scheduler: one-shot poll failed", "error", err)
	}
	s.checkOverdueBackup()
}

func (s *Scheduler) loadSchedules() error {
	db, err := s.openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	schedules, err := scheduler.ListEnabled(db)
	if err != nil {
		return fmt.Errorf("list enabled: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	activeIDs := make(map[int64]bool)
	for _, sched := range schedules {
		if sched.Type != scheduler.Recurring || sched.CronExpr == "" {
			continue
		}
		if !s.isValidFrequency(sched.CronExpr) {
			slog.Warn("scheduler: skipping schedule with frequency < 1 hour", "id", sched.ID)
			continue
		}
		activeIDs[sched.ID] = true
		if _, exists := s.entries[sched.ID]; exists {
			continue
		}
		entryID, err := s.addCronEntry(sched)
		if err != nil {
			slog.Error("scheduler: add cron entry", "id", sched.ID, "error", err)
			continue
		}
		s.entries[sched.ID] = entryID
		slog.Info("scheduler: added cron entry", "id", sched.ID, "cron", sched.CronExpr)
	}

	for id, entryID := range s.entries {
		if !activeIDs[id] {
			s.cron.Remove(entryID)
			delete(s.entries, id)
			slog.Info("scheduler: removed cron entry", "id", id)
		}
	}

	return nil
}

func (s *Scheduler) addCronEntry(sched scheduler.Schedule) (cron.EntryID, error) {
	schedID := sched.ID
	return s.cron.AddFunc(sched.CronExpr, func() {
		db, err := s.openDB()
		if err != nil {
			slog.Error("scheduler: cron open db", "error", err)
			return
		}
		defer db.Close()

		fresh, err := scheduler.Get(db, schedID)
		if err != nil {
			slog.Error("scheduler: cron re-read schedule", "schedule_id", schedID, "error", err)
			return
		}
		if fresh.LastRunAt != nil && time.Since(*fresh.LastRunAt) < 5*time.Minute {
			slog.Debug("scheduler: cron skip (recent stamp)", "schedule_id", schedID)
			return
		}

		if err := s.enqueueScheduledTask(*fresh); err != nil {
			slog.Error("scheduler: cron enqueue", "schedule_id", schedID, "error", err)
			return
		}
		if err := scheduler.UpdateLastRun(db, schedID, time.Now().UTC(), ""); err != nil {
			slog.Error("scheduler: cron stamp", "schedule_id", schedID, "error", err)
		}
	})
}

func (s *Scheduler) pollOneShot(ctx context.Context) error {
	db, err := s.openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	due, err := scheduler.ListDueOneShot(db, time.Now().UTC())
	if err != nil {
		return err
	}

	for _, sched := range due {
		if err := s.enqueueScheduledTask(sched); err != nil {
			slog.Error("scheduler: one-shot enqueue", "schedule_id", sched.ID, "error", err)
			continue
		}

		if err := scheduler.Disable(db, sched.ID); err != nil {
			slog.Error("scheduler: disable one-shot after enqueue", "schedule_id", sched.ID, "error", err)
		}

		slog.Info("scheduler: enqueued one-shot task", "schedule_id", sched.ID)
	}

	return nil
}

func (s *Scheduler) reactiveCheckLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case sig := <-s.notifier.C():
			if err := s.checkReactiveTriggers(ctx, sig.Source); err != nil {
				slog.Error("scheduler: reactive check failed", "source", sig.Source, "error", err)
			}
		}
	}
}

func (s *Scheduler) checkReactiveTriggers(ctx context.Context, source string) error {
	schedDB, err := s.openDB()
	if err != nil {
		return fmt.Errorf("open scheduler db: %w", err)
	}
	defer schedDB.Close()

	schedules, err := scheduler.ListEnabledReactive(schedDB, source)
	if err != nil {
		return fmt.Errorf("list reactive: %w", err)
	}
	if len(schedules) == 0 {
		return nil
	}

	dsn, err := s.cfg.SourceDataDSN(source)
	if err != nil {
		return fmt.Errorf("source dsn: %w", err)
	}
	sourceDB, err := store.Open(store.SQLiteConfig(dsn))
	if err != nil {
		return fmt.Errorf("open source db %q: %w", source, err)
	}
	defer sourceDB.Close()

	return s.checkReactiveTriggersWithDB(ctx, schedDB, sourceDB, schedules)
}

func (s *Scheduler) checkReactiveTriggersWithDB(ctx context.Context, schedDB *store.DB, sourceDB *store.DB, schedules []scheduler.Schedule) error {
	for _, sched := range schedules {
		match, err := scheduler.CheckTrigger(sourceDB, sched.TriggerSource, sched.TriggerQuery, sched.LastTriggerID)
		if err != nil {
			slog.Error("scheduler: trigger check failed", "id", sched.ID, "error", err)
			continue
		}
		if match == nil {
			continue
		}

		// Build augmented task with matched data summary.
		task := sched.Task + "\n\nTriggered by " + fmt.Sprintf("%d", len(match.Rows)) + " new matching row(s) from " + sched.TriggerSource + ":\n"
		for i, row := range match.Rows {
			if i >= 5 {
				task += fmt.Sprintf("... and %d more\n", len(match.Rows)-5)
				break
			}
			task += formatRow(row)
		}

		augmented := sched
		augmented.Task = task
		if err := s.enqueueScheduledTask(augmented); err != nil {
			slog.Error("scheduler: reactive enqueue", "schedule_id", sched.ID, "error", err)
			continue
		}

		if err := scheduler.UpdateLastTriggerID(schedDB, sched.ID, match.MaxID); err != nil {
			slog.Error("scheduler: update watermark", "id", sched.ID, "error", err)
		}
		if err := scheduler.UpdateLastRun(schedDB, sched.ID, time.Now().UTC(), ""); err != nil {
			slog.Error("scheduler: update last run", "id", sched.ID, "error", err)
		}

		slog.Info("scheduler: enqueued reactive task", "schedule_id", sched.ID, "matched_rows", len(match.Rows), "watermark", match.MaxID)
	}
	return nil
}

// CheckReactiveTriggersForTest exposes reactive trigger checking for tests.
func (s *Scheduler) CheckReactiveTriggersForTest(ctx context.Context, source string, sourceDB *store.DB) error {
	schedDB, err := s.openDB()
	if err != nil {
		return fmt.Errorf("open scheduler db: %w", err)
	}
	defer schedDB.Close()

	schedules, err := scheduler.ListEnabledReactive(schedDB, source)
	if err != nil {
		return fmt.Errorf("list reactive: %w", err)
	}
	return s.checkReactiveTriggersWithDB(ctx, schedDB, sourceDB, schedules)
}

func formatRow(row map[string]string) string {
	keys := make([]string, 0, len(row))
	for k := range row {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+": "+row[k])
	}
	return "  " + strings.Join(parts, " | ") + "\n"
}

func (s *Scheduler) checkOverdueRecurring() {
	db, err := s.openDB()
	if err != nil {
		slog.Error("scheduler: overdue-recurring open db", "error", err)
		return
	}
	defer db.Close()

	schedules, err := scheduler.ListEnabled(db)
	if err != nil {
		slog.Error("scheduler: overdue-recurring list", "error", err)
		return
	}

	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	now := time.Now().UTC()

	for _, sched := range schedules {
		if sched.Type != scheduler.Recurring || sched.CronExpr == "" {
			continue
		}
		if sched.LastRunAt == nil {
			continue
		}

		cronSched, err := parser.Parse(sched.CronExpr)
		if err != nil {
			continue
		}

		nextExpected := cronSched.Next(*sched.LastRunAt)
		if !nextExpected.Before(now) {
			continue
		}

		slog.Info("scheduler: recurring overdue", "id", sched.ID, "last_run", sched.LastRunAt, "was_due", nextExpected)

		if err := s.enqueueScheduledTask(sched); err != nil {
			slog.Error("scheduler: overdue-recurring enqueue", "schedule_id", sched.ID, "error", err)
			continue
		}

		if err := scheduler.UpdateLastRun(db, sched.ID, now, ""); err != nil {
			slog.Error("scheduler: overdue-recurring stamp", "schedule_id", sched.ID, "error", err)
		}
	}
}

func (s *Scheduler) checkOverdueBackup() {
	if s.cfg.Backup == nil || !s.cfg.Backup.Enabled || s.cfg.Backup.Schedule == "" {
		return
	}
	if !config.IsSourceLinked("backup") {
		return
	}

	schedule, err := time.ParseDuration(s.cfg.Backup.Schedule)
	if err != nil || schedule <= 0 {
		return
	}

	manifest, err := backupsvc.LoadManifest(config.BackupLastManifestPath())
	if err != nil {
		slog.Error("scheduler: backup manifest load", "error", err)
		return
	}

	if manifest.ID != "" && time.Since(manifest.Timestamp) < schedule {
		return
	}

	slog.Info("scheduler: backup overdue", "last", manifest.Timestamp, "schedule", schedule)

	tx, err := s.jobsDB.Begin()
	if err != nil {
		slog.Error("scheduler: backup begin tx", "error", err)
		return
	}
	_, err = s.river.InsertTx(s.ctx, tx, jobs.BackupArgs{}, &river.InsertOpts{
		MaxAttempts: 3,
		UniqueOpts: river.UniqueOpts{
			ByPeriod: schedule,
		},
	})
	if err != nil {
		tx.Rollback()
		slog.Error("scheduler: backup insert", "error", err)
		return
	}
	if err := tx.Commit(); err != nil {
		slog.Error("scheduler: backup commit", "error", err)
	}
}

func (s *Scheduler) enqueueScheduledTask(sched scheduler.Schedule) error {
	metaJSON, _ := json.Marshal(sched.ChannelMeta)
	args := jobs.ScheduledTaskArgs{
		ScheduleID:   sched.ID,
		Task:         sched.Task,
		Channel:      sched.Channel,
		ChannelMeta:  string(metaJSON),
		ModelTier:    sched.ModelTier,
		MaxBudgetUSD: sched.MaxBudgetUSD,
	}

	tx, err := s.jobsDB.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	_, err = s.river.InsertTx(s.ctx, tx, args, &river.InsertOpts{
		MaxAttempts: 3,
	})
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("insert: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func (s *Scheduler) isValidFrequency(cronExpr string) bool {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	sched, err := parser.Parse(cronExpr)
	if err != nil {
		return false
	}
	now := time.Now().UTC()
	first := sched.Next(now)
	second := sched.Next(first)
	return second.Sub(first) >= time.Hour
}

func (s *Scheduler) openDB() (*store.DB, error) {
	return store.Open(store.Config{
		Driver: s.cfg.Scheduler.Storage.Driver,
		DSN:    s.cfg.SchedulerDataDSN(),
	})
}
