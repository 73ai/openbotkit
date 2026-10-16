package schema

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/73ai/openbotkit/config"
	contactsrc "github.com/73ai/openbotkit/service/contacts"
	schedsrc "github.com/73ai/openbotkit/service/scheduler"
	ansrc "github.com/73ai/openbotkit/source/applenotes"
	gmailsrc "github.com/73ai/openbotkit/source/gmail"
	imsrc "github.com/73ai/openbotkit/source/imessage"
	wasrc "github.com/73ai/openbotkit/source/whatsapp"
	"github.com/73ai/openbotkit/store"
)

func MigrateAll(cfg *config.Config) error {
	migrations := []struct {
		name   string
		driver string
		dsn    string
		fn     func(*store.DB) error
	}{
		{"applenotes", cfg.AppleNotes.Storage.Driver, cfg.AppleNotesDataDSN(), ansrc.Migrate},
		{"imessage", cfg.IMessage.Storage.Driver, cfg.IMessageDataDSN(), imsrc.Migrate},
		{"whatsapp", cfg.WhatsApp.Storage.Driver, cfg.WhatsAppDataDSN(), wasrc.Migrate},
		{"gmail", cfg.Gmail.Storage.Driver, cfg.GmailDataDSN(), gmailsrc.Migrate},
		{"contacts", cfg.Contacts.Storage.Driver, cfg.ContactsDataDSN(), contactsrc.Migrate},
		{"scheduler", cfg.Scheduler.Storage.Driver, cfg.SchedulerDataDSN(), schedsrc.Migrate},
	}
	for _, m := range migrations {
		if m.driver == "sqlite" {
			os.MkdirAll(filepath.Dir(m.dsn), 0700)
		}
		db, err := store.Open(store.Config{Driver: m.driver, DSN: m.dsn})
		if err != nil {
			slog.Warn("migrate: open failed", "source", m.name, "error", err)
			continue
		}
		err = m.fn(db)
		db.Close()
		if errors.Is(err, store.ErrSchemaTooNew) {
			return err
		}
		if err != nil {
			slog.Warn("migrate: failed", "source", m.name, "error", err)
		}
	}
	return nil
}
