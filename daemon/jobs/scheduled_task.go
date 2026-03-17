package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/riverqueue/river"

	"github.com/73ai/openbotkit/agent"
	"github.com/73ai/openbotkit/agent/audit"
	"github.com/73ai/openbotkit/agent/tools"
	"github.com/73ai/openbotkit/channel"
	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/provider"
	"github.com/73ai/openbotkit/source/scheduler"
	"github.com/73ai/openbotkit/store"
)

type ScheduledTaskArgs struct {
	ScheduleID  int64  `json:"schedule_id"`
	Task        string `json:"task"`
	Channel     string `json:"channel"`
	ChannelMeta string `json:"channel_meta"`
}

func (ScheduledTaskArgs) Kind() string { return "scheduled_task" }

// PusherFactory builds a Pusher for delivering results. If nil, the default
// channel.NewPusher is used.
type PusherFactory func(channelType string, meta scheduler.ChannelMeta) (channel.Pusher, error)

// AgentRunner executes a task prompt and returns the result text.
// If nil, the default LLM-based agent is used.
type AgentRunner func(ctx context.Context, task string) (string, error)

type ScheduledTaskWorker struct {
	river.WorkerDefaults[ScheduledTaskArgs]
	Cfg           *config.Config
	MakePusher    PusherFactory
	RunAgentFunc  AgentRunner
}

func (w *ScheduledTaskWorker) Work(ctx context.Context, job *river.Job[ScheduledTaskArgs]) error {
	slog.Info("running scheduled task", "schedule_id", job.Args.ScheduleID, "attempt", job.Attempt)

	var meta scheduler.ChannelMeta
	if err := json.Unmarshal([]byte(job.Args.ChannelMeta), &meta); err != nil {
		return fmt.Errorf("parse channel meta: %w", err)
	}

	runAgent := w.runAgent
	if w.RunAgentFunc != nil {
		runAgent = w.RunAgentFunc
	}
	result, err := runAgent(ctx, job.Args.Task)
	if err != nil {
		slog.Error("scheduled task agent failed", "schedule_id", job.Args.ScheduleID, "error", err)
		w.updateLastRun(job.Args.ScheduleID, err.Error())

		if job.Attempt >= 2 {
			w.notifyFailure(ctx, job.Args.Channel, meta, job.Args.ScheduleID, err)
			return nil
		}
		return err
	}

	pusher, err := w.newPusher(job.Args.Channel, meta)
	if err != nil {
		slog.Error("scheduled task: create pusher", "schedule_id", job.Args.ScheduleID, "error", err)
		w.updateLastRun(job.Args.ScheduleID, fmt.Sprintf("create pusher: %v", err))
		return fmt.Errorf("create pusher: %w", err)
	}
	if err := pusher.Push(ctx, result); err != nil {
		slog.Error("scheduled task: push result", "schedule_id", job.Args.ScheduleID, "error", err)
		w.updateLastRun(job.Args.ScheduleID, fmt.Sprintf("push: %v", err))
		return fmt.Errorf("push result: %w", err)
	}

	w.updateLastRun(job.Args.ScheduleID, "")
	w.maybeMarkCompleted(job.Args.ScheduleID)

	slog.Info("scheduled task complete", "schedule_id", job.Args.ScheduleID)
	return nil
}

func (w *ScheduledTaskWorker) NextRetryAt(_ *river.Job[ScheduledTaskArgs]) time.Time {
	return time.Now().Add(15 * time.Minute)
}

func (w *ScheduledTaskWorker) runAgent(ctx context.Context, task string) (string, error) {
	if w.Cfg == nil || w.Cfg.Models == nil || w.Cfg.Models.Default == "" {
		return "", fmt.Errorf("no LLM model configured")
	}

	registry, err := provider.NewRegistry(w.Cfg.Models)
	if err != nil {
		return "", fmt.Errorf("create provider registry: %w", err)
	}

	providerName, modelName, err := provider.ParseModelSpec(w.Cfg.Models.Default)
	if err != nil {
		return "", fmt.Errorf("parse model spec: %w", err)
	}

	p, ok := registry.Get(providerName)
	if !ok {
		return "", fmt.Errorf("provider %q not found", providerName)
	}

	toolReg := tools.NewScheduledTaskRegistry()
	sessionID := fmt.Sprintf("sched-%d", time.Now().UnixMilli())
	if err := config.EnsureScratchDir(sessionID); err != nil {
		slog.Warn("scratch dir creation failed", "error", err)
	}
	toolReg.SetScratchDir(config.ScratchDir(sessionID))
	defer config.CleanScratch(sessionID)

	al := openAuditLogger()
	if al != nil {
		defer al.Close()
		toolReg.SetAudit(al, "scheduled")
	}

	identity := "You are a scheduled task agent. Execute the task and return a concise result.\n"
	blocks := tools.BuildSystemBlocks(identity, toolReg)

	a := agent.New(p, modelName, toolReg, agent.WithSystemBlocks(blocks))
	return a.Run(ctx, task)
}

func (w *ScheduledTaskWorker) updateLastRun(scheduleID int64, lastErr string) {
	db, err := store.Open(store.Config{
		Driver: w.Cfg.Scheduler.Storage.Driver,
		DSN:    w.Cfg.SchedulerDataDSN(),
	})
	if err != nil {
		slog.Error("scheduled task: open scheduler db", "error", err)
		return
	}
	defer db.Close()

	if err := scheduler.UpdateLastRun(db, scheduleID, time.Now().UTC(), lastErr); err != nil {
		slog.Error("scheduled task: update last run", "error", err)
	}
}

func (w *ScheduledTaskWorker) maybeMarkCompleted(scheduleID int64) {
	db, err := store.Open(store.Config{
		Driver: w.Cfg.Scheduler.Storage.Driver,
		DSN:    w.Cfg.SchedulerDataDSN(),
	})
	if err != nil {
		return
	}
	defer db.Close()

	s, err := scheduler.Get(db, scheduleID)
	if err != nil {
		return
	}
	if s.Type == scheduler.OneShot {
		_ = scheduler.MarkCompleted(db, scheduleID, time.Now().UTC())
	}
}

func (w *ScheduledTaskWorker) newPusher(channelType string, meta scheduler.ChannelMeta) (channel.Pusher, error) {
	if w.MakePusher != nil {
		return w.MakePusher(channelType, meta)
	}
	return channel.NewPusher(channelType, meta)
}

func (w *ScheduledTaskWorker) notifyFailure(ctx context.Context, ch string, meta scheduler.ChannelMeta, scheduleID int64, taskErr error) {
	pusher, err := w.newPusher(ch, meta)
	if err != nil {
		slog.Error("scheduled task: create failure pusher", "error", err)
		return
	}
	msg := fmt.Sprintf("Scheduled task #%d failed after retries: %v", scheduleID, taskErr)
	if err := pusher.Push(ctx, msg); err != nil {
		slog.Error("scheduled task: push failure notification", "error", err)
	}
}

func openAuditLogger() *audit.Logger {
	return audit.OpenDefault(config.AuditDBPath())
}

var _ river.Worker[ScheduledTaskArgs] = (*ScheduledTaskWorker)(nil)
