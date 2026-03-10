package daemon

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/riverqueue/river"

	"github.com/priyanshujain/openbotkit/config"
)

type Daemon struct {
	cfg            *config.Config
	river          *river.Client[*sql.Tx]
	jobsDB         *sql.DB
	skipAppleNotes bool
	skipWhatsApp   bool
	skipIMessage   bool
}

type Option func(*Daemon)

func WithSkipAppleNotes() Option {
	return func(d *Daemon) { d.skipAppleNotes = true }
}

func WithSkipWhatsApp() Option {
	return func(d *Daemon) { d.skipWhatsApp = true }
}

func WithSkipIMessage() Option {
	return func(d *Daemon) { d.skipIMessage = true }
}

func New(cfg *config.Config, opts ...Option) *Daemon {
	d := &Daemon{cfg: cfg}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

func (d *Daemon) Run(ctx context.Context) error {
	if err := config.EnsureDir(); err != nil {
		return fmt.Errorf("ensure config dir: %w", err)
	}

	lock, err := acquireLock()
	if err != nil {
		return err
	}
	defer releaseLock(lock)

	slog.Info("starting daemon")

	client, db, err := newRiverClient(ctx, d.cfg)
	if err != nil {
		return fmt.Errorf("init river: %w", err)
	}
	d.river = client
	d.jobsDB = db

	if err := d.river.Start(ctx); err != nil {
		d.jobsDB.Close()
		return fmt.Errorf("start river: %w", err)
	}
	slog.Info("river job queue started")

	var waErrCh, anErrCh, imErrCh <-chan error
	if !d.skipWhatsApp {
		waErrCh = runWhatsAppSync(ctx, d.cfg)
	}
	if !d.skipAppleNotes {
		anErrCh = runAppleNotesSync(ctx, d.cfg)
	}
	if !d.skipIMessage {
		imErrCh = runIMessageSync(ctx, d.cfg)
	}

	// Block until context is cancelled (signal received).
	<-ctx.Done()
	slog.Info("shutting down daemon")

	// Drain sync errors.
	if waErrCh != nil {
		if err := <-waErrCh; err != nil {
			slog.Error("whatsapp error during shutdown", "error", err)
		}
	}
	if anErrCh != nil {
		if err := <-anErrCh; err != nil {
			slog.Error("applenotes error during shutdown", "error", err)
		}
	}
	if imErrCh != nil {
		if err := <-imErrCh; err != nil {
			slog.Error("imessage error during shutdown", "error", err)
		}
	}

	if err := d.river.Stop(context.Background()); err != nil {
		slog.Error("river stop error", "error", err)
	}
	d.jobsDB.Close()

	slog.Info("daemon stopped")
	return nil
}
