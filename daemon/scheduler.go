package daemon

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/riverqueue/river"
	"github.com/robfig/cron/v3"

	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/daemon/jobs"
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

	if err := s.loadSchedules(); err != nil {
		slog.Error("scheduler: initial load failed", "error", err)
	}

	go s.reloadLoop(ctx)
	go s.oneShotLoop(ctx)
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

func (s *Scheduler) reloadLoop(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.loadSchedules(); err != nil {
				slog.Error("scheduler: reload failed", "error", err)
			}
		}
	}
}

func (s *Scheduler) oneShotLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.pollOneShot(ctx); err != nil {
				slog.Error("scheduler: one-shot poll failed", "error", err)
			}
		}
	}
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
	metaJSON, _ := json.Marshal(sched.ChannelMeta)
	args := jobs.ScheduledTaskArgs{
		ScheduleID:   sched.ID,
		Task:         sched.Task,
		Channel:      sched.Channel,
		ChannelMeta:  string(metaJSON),
		ModelTier:    sched.ModelTier,
		MaxBudgetUSD: sched.MaxBudgetUSD,
	}

	return s.cron.AddFunc(sched.CronExpr, func() {
		tx, err := s.jobsDB.Begin()
		if err != nil {
			slog.Error("scheduler: begin tx", "error", err)
			return
		}
		_, err = s.river.InsertTx(s.ctx, tx, args, &river.InsertOpts{
			MaxAttempts: 3,
		})
		if err != nil {
			tx.Rollback()
			slog.Error("scheduler: insert job", "schedule_id", sched.ID, "error", err)
			return
		}
		if err := tx.Commit(); err != nil {
			slog.Error("scheduler: commit tx", "error", err)
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
			slog.Error("scheduler: begin tx for one-shot", "error", err)
			continue
		}
		_, err = s.river.InsertTx(ctx, tx, args, &river.InsertOpts{
			MaxAttempts: 3,
		})
		if err != nil {
			tx.Rollback()
			slog.Error("scheduler: insert one-shot job", "schedule_id", sched.ID, "error", err)
			continue
		}
		if err := tx.Commit(); err != nil {
			slog.Error("scheduler: commit one-shot tx", "error", err)
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
			task += fmt.Sprintf("  %v\n", row)
		}

		metaJSON, _ := json.Marshal(sched.ChannelMeta)
		args := jobs.ScheduledTaskArgs{
			ScheduleID:   sched.ID,
			Task:         task,
			Channel:      sched.Channel,
			ChannelMeta:  string(metaJSON),
			ModelTier:    sched.ModelTier,
			MaxBudgetUSD: sched.MaxBudgetUSD,
		}

		tx, err := s.jobsDB.Begin()
		if err != nil {
			slog.Error("scheduler: begin tx for reactive", "error", err)
			continue
		}
		_, err = s.river.InsertTx(ctx, tx, args, &river.InsertOpts{
			MaxAttempts: 3,
		})
		if err != nil {
			tx.Rollback()
			slog.Error("scheduler: insert reactive job", "schedule_id", sched.ID, "error", err)
			continue
		}
		if err := tx.Commit(); err != nil {
			slog.Error("scheduler: commit reactive tx", "error", err)
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
