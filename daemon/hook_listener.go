package daemon

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/riverqueue/river"

	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/daemon/jobs"
	"github.com/73ai/openbotkit/service/hooks"
	"github.com/73ai/openbotkit/store"
)

const defaultGmailHookPrompt = `You will receive an email. Decide if it is urgent and requires immediate attention.

If NOT urgent, reply with exactly: SKIP
If urgent, write a short Telegram notification (1-3 sentences) summarizing the email and why it's urgent.

Urgent examples: production outages, security alerts, time-sensitive deadlines, emergency contacts.
Not urgent: newsletters, marketing, routine updates, automated notifications.`

type HookListener struct {
	cfg      *config.Config
	river    *river.Client[*sql.Tx]
	jobsDB   *sql.DB
	notifier *SyncNotifier
	hooksDB  *store.DB
	ch       <-chan SyncSignal
}

func NewHookListener(cfg *config.Config, rc *river.Client[*sql.Tx], jobsDB *sql.DB, notifier *SyncNotifier, hooksDB *store.DB) *HookListener {
	return &HookListener{
		cfg:      cfg,
		river:    rc,
		jobsDB:   jobsDB,
		notifier: notifier,
		hooksDB:  hooksDB,
		ch:       notifier.Subscribe(), // subscribe eagerly so no signals are lost
	}
}

func (l *HookListener) Run(ctx context.Context) {
	l.ensureDefaults()
	ch := l.ch

	for {
		select {
		case <-ctx.Done():
			return
		case sig := <-ch:
			l.handleEvent(ctx, sig)
		}
	}
}

func (l *HookListener) handleEvent(ctx context.Context, sig SyncSignal) {
	enabled, err := hooks.ListEnabled(l.hooksDB, sig.Source+"_sync")
	if err != nil {
		slog.Warn("hook: list failed", "source", sig.Source, "err", err)
		return
	}
	if len(enabled) == 0 {
		return
	}

	ids, ok := sig.Data.([]int64)
	if !ok || len(ids) == 0 {
		return
	}

	for _, hook := range enabled {
		tx, err := l.jobsDB.Begin()
		if err != nil {
			slog.Warn("hook: begin tx failed", "err", err)
			continue
		}
		_, err = l.river.InsertTx(ctx, tx, jobs.EventHookArgs{
			HookID:  hook.ID,
			ItemIDs: ids,
		}, nil)
		if err != nil {
			tx.Rollback()
			slog.Warn("hook: enqueue failed", "hook_id", hook.ID, "err", err)
			continue
		}
		if err := tx.Commit(); err != nil {
			slog.Warn("hook: commit failed", "hook_id", hook.ID, "err", err)
		}
	}
}

func (l *HookListener) ensureDefaults() {
	if !config.IsSourceLinked("gmail") {
		return
	}
	existing, err := hooks.ListEnabled(l.hooksDB, "gmail_sync")
	if err != nil {
		slog.Warn("hook: list defaults failed", "err", err)
		return
	}
	if len(existing) > 0 {
		return
	}
	id, err := hooks.Create(l.hooksDB, &hooks.EventHook{
		EventType: "gmail_sync",
		Prompt:    defaultGmailHookPrompt,
		Channel:   "telegram",
		ModelTier: "nano",
	})
	if err != nil {
		slog.Warn("hook: auto-create failed", "err", err)
		return
	}
	slog.Info("hook: created default gmail_sync hook", "id", id)
}
