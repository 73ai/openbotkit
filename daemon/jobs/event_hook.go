package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/riverqueue/river"

	"github.com/73ai/openbotkit/channel"
	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/provider"
	"github.com/73ai/openbotkit/service/hooks"
	gmailsrc "github.com/73ai/openbotkit/source/gmail"
	"github.com/73ai/openbotkit/store"
)

type EventHookArgs struct {
	HookID  int64   `json:"hook_id"`
	ItemIDs []int64 `json:"item_ids"`
}

func (EventHookArgs) Kind() string { return "event_hook" }

type EventHookWorker struct {
	river.WorkerDefaults[EventHookArgs]
	Cfg     *config.Config
	ChanReg *channel.Registry
	HooksDB *store.DB
}

func (w *EventHookWorker) Work(ctx context.Context, job *river.Job[EventHookArgs]) error {
	hook, err := hooks.Get(w.HooksDB, job.Args.HookID)
	if err != nil {
		return fmt.Errorf("load hook %d: %w", job.Args.HookID, err)
	}
	if !hook.Enabled {
		slog.Info("hook disabled, skipping", "hook_id", hook.ID)
		return nil
	}

	emails, err := w.loadEmails(job.Args.ItemIDs)
	if err != nil {
		return fmt.Errorf("load emails: %w", err)
	}
	if len(emails) == 0 {
		return nil
	}

	p, model, err := w.resolveProvider(hook.ModelTier)
	if err != nil {
		return fmt.Errorf("resolve provider: %w", err)
	}

	pusher, err := w.ChanReg.Get(hook.Channel)
	if err != nil {
		return fmt.Errorf("get pusher: %w", err)
	}

	for _, email := range emails {
		prompt := hook.Prompt + "\n\nFrom: " + email.From +
			"\nSubject: " + email.Subject +
			"\nDate: " + email.Date.Format("2006-01-02 15:04") +
			"\nBody: " + truncate(email.Body, 500)

		resp, err := p.Chat(ctx, provider.ChatRequest{
			Model: model,
			Messages: []provider.Message{
				{Role: "user", Content: []provider.ContentBlock{{Type: "text", Text: prompt}}},
			},
		})
		if err != nil {
			slog.Warn("hook: classify failed", "email_id", email.MessageID, "err", err)
			continue
		}

		text := strings.TrimSpace(resp.TextContent())
		if strings.EqualFold(text, "SKIP") {
			continue
		}
		if err := pusher.Push(ctx, text); err != nil {
			slog.Warn("hook: push failed", "email_id", email.MessageID, "err", err)
		}
	}
	return nil
}

func (w *EventHookWorker) loadEmails(ids []int64) ([]gmailsrc.Email, error) {
	db, err := store.Open(store.Config{
		Driver: w.Cfg.Gmail.Storage.Driver,
		DSN:    w.Cfg.GmailDataDSN(),
	})
	if err != nil {
		return nil, fmt.Errorf("open gmail db: %w", err)
	}
	defer db.Close()
	return gmailsrc.GetEmailsByIDs(db, ids)
}

func (w *EventHookWorker) resolveProvider(modelTier string) (provider.Provider, string, error) {
	if w.Cfg.Models == nil {
		return nil, "", fmt.Errorf("no models configured")
	}
	registry, err := provider.NewRegistry(w.Cfg.Models)
	if err != nil {
		return nil, "", fmt.Errorf("create provider registry: %w", err)
	}
	router := provider.NewRouter(registry, w.Cfg.Models)
	tier := provider.TierNano
	if modelTier != "" {
		tier = provider.ModelTier(modelTier)
	}
	return router.Resolve(tier)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

var _ river.Worker[EventHookArgs] = (*EventHookWorker)(nil)
