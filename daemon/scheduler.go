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
	cfg     *config.Config
	river   *river.Client[*sql.Tx]
	jobsDB  *sql.DB
	cron    *cron.Cron
	mu      sync.Mutex
	entries map[int64]cron.EntryID
	ctx     context.Context
}

func NewScheduler(cfg *config.Config, riverClient *river.Client[*sql.Tx], jobsDB *sql.DB) *Scheduler {
	return &Scheduler{
		cfg:     cfg,
		river:   riverClient,
		jobsDB:  jobsDB,
		entries: make(map[int64]cron.EntryID),
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
		ScheduleID:  sched.ID,
		Task:        sched.Task,
		Channel:     sched.Channel,
		ChannelMeta: string(metaJSON),
	}

	return s.cron.AddFunc(sched.CronExpr, func() {
		tx, err := s.jobsDB.Begin()
		if err != nil {
			slog.Error("scheduler: begin tx", "error", err)
			return
		}
		_, err = s.river.InsertTx(s.ctx, tx, args, &river.InsertOpts{
			MaxAttempts: 2,
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
			ScheduleID:  sched.ID,
			Task:        sched.Task,
			Channel:     sched.Channel,
			ChannelMeta: string(metaJSON),
		}

		tx, err := s.jobsDB.Begin()
		if err != nil {
			slog.Error("scheduler: begin tx for one-shot", "error", err)
			continue
		}
		_, err = s.river.InsertTx(ctx, tx, args, &river.InsertOpts{
			MaxAttempts: 2,
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
