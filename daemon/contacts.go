package daemon

import (
	"context"
	"log/slog"
	"runtime"
	"time"

	"github.com/73ai/openbotkit/config"
	contactsrc "github.com/73ai/openbotkit/source/contacts"
	"github.com/73ai/openbotkit/store"
)

const contactsSyncInterval = 5 * time.Minute

func runContactsSync(ctx context.Context, cfg *config.Config) <-chan error {
	errCh := make(chan error, 1)

	go func() {
		defer close(errCh)

		migrateContactsLinking()

		if err := config.EnsureSourceDir("contacts"); err != nil {
			slog.Error("contacts: failed to create dir", "error", err)
			errCh <- err
			return
		}

		db, err := store.Open(store.Config{
			Driver: cfg.Contacts.Storage.Driver,
			DSN:    cfg.ContactsDataDSN(),
		})
		if err != nil {
			slog.Error("contacts: failed to open db", "error", err)
			errCh <- err
			return
		}
		defer db.Close()

		sourceDBs := openSourceDBs(cfg)
		defer closeSourceDBs(sourceDBs)

		if len(sourceDBs) == 0 && !config.IsSourceLinked("applecontacts") {
			slog.Info("contacts: no linked sources, skipping sync")
			return
		}

		syncContacts(db, sourceDBs)

		ticker := time.NewTicker(contactsSyncInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				slog.Info("contacts: stopping sync")
				return
			case <-ticker.C:
				syncContacts(db, sourceDBs)
			}
		}
	}()

	return errCh
}

func linkedSources(sourceDBs map[string]*store.DB) []string {
	sources := make([]string, 0, len(sourceDBs)+1)
	for name := range sourceDBs {
		sources = append(sources, name)
	}
	if config.IsSourceLinked("applecontacts") {
		sources = append(sources, "applecontacts")
	}
	return sources
}

func syncContacts(db *store.DB, sourceDBs map[string]*store.DB) {
	result, err := contactsrc.Sync(db, sourceDBs, contactsrc.SyncOptions{
		Sources: linkedSources(sourceDBs),
	})
	if err != nil {
		slog.Error("contacts: sync error", "error", err)
		return
	}
	slog.Info("contacts: sync complete",
		"created", result.Created, "linked", result.Linked, "errors", result.Errors)
}

func openSourceDBs(cfg *config.Config) map[string]*store.DB {
	dbs := make(map[string]*store.DB)
	sources := []struct {
		name   string
		driver string
		dsn    string
	}{
		{"whatsapp", cfg.WhatsApp.Storage.Driver, cfg.WhatsAppDataDSN()},
		{"gmail", cfg.Gmail.Storage.Driver, cfg.GmailDataDSN()},
		{"imessage", cfg.IMessage.Storage.Driver, cfg.IMessageDataDSN()},
	}
	for _, s := range sources {
		if !config.IsSourceLinked(s.name) {
			continue
		}
		sdb, err := store.Open(store.Config{Driver: s.driver, DSN: s.dsn})
		if err != nil {
			slog.Warn("contacts: could not open source db", "source", s.name, "error", err)
			continue
		}
		dbs[s.name] = sdb
	}
	return dbs
}

func closeSourceDBs(dbs map[string]*store.DB) {
	for _, db := range dbs {
		db.Close()
	}
}

func migrateContactsLinking() {
	if runtime.GOOS != "darwin" {
		return
	}
	if config.IsSourceLinked("contacts") && !config.IsSourceLinked("applecontacts") {
		if err := config.LinkSource("applecontacts"); err != nil {
			slog.Warn("contacts: failed to migrate contacts->applecontacts linking", "error", err)
		} else {
			slog.Info("contacts: migrated contacts->applecontacts linking")
		}
	}
}
