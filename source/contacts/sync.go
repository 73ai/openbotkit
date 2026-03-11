package contacts

import (
	"fmt"
	"log/slog"
	"runtime"
	"slices"

	"github.com/priyanshujain/openbotkit/store"
)

func Sync(contactsDB *store.DB, sourceDBs map[string]*store.DB, opts SyncOptions) (*SyncResult, error) {
	if err := Migrate(contactsDB); err != nil {
		return nil, fmt.Errorf("migrate contacts schema: %w", err)
	}

	total := &SyncResult{}

	if shouldSync(opts, "whatsapp") {
		if waDB, ok := sourceDBs["whatsapp"]; ok {
			slog.Info("contacts: syncing from whatsapp")
			r, err := syncFromWhatsApp(contactsDB, waDB)
			if err != nil {
				slog.Error("contacts: whatsapp sync failed", "error", err)
				total.Errors++
			} else {
				merge(total, r)
				if err := SaveSyncState(contactsDB, "whatsapp", ""); err != nil {
					slog.Warn("contacts: save whatsapp sync state", "error", err)
				}
			}
		}
	}

	if shouldSync(opts, "gmail") {
		if gmailDB, ok := sourceDBs["gmail"]; ok {
			slog.Info("contacts: syncing from gmail")
			r, err := syncFromGmail(contactsDB, gmailDB)
			if err != nil {
				slog.Error("contacts: gmail sync failed", "error", err)
				total.Errors++
			} else {
				merge(total, r)
				if err := SaveSyncState(contactsDB, "gmail", ""); err != nil {
					slog.Warn("contacts: save gmail sync state", "error", err)
				}
			}
		}
	}

	if shouldSync(opts, "imessage") {
		if imDB, ok := sourceDBs["imessage"]; ok {
			slog.Info("contacts: syncing from imessage")
			r, err := syncFromIMessage(contactsDB, imDB)
			if err != nil {
				slog.Error("contacts: imessage sync failed", "error", err)
				total.Errors++
			} else {
				merge(total, r)
				if err := SaveSyncState(contactsDB, "imessage", ""); err != nil {
					slog.Warn("contacts: save imessage sync state", "error", err)
				}
			}
		}
	}

	if shouldSync(opts, "applecontacts") && runtime.GOOS == "darwin" {
		slog.Info("contacts: syncing from apple contacts")
		r, err := syncFromAppleContacts(contactsDB)
		if err != nil {
			slog.Error("contacts: apple contacts sync failed", "error", err)
			total.Errors++
		} else {
			merge(total, r)
			if err := SaveSyncState(contactsDB, "applecontacts", ""); err != nil {
				slog.Warn("contacts: save applecontacts sync state", "error", err)
			}
		}
	}

	slog.Info("contacts: sync complete",
		"created", total.Created, "updated", total.Updated,
		"linked", total.Linked, "errors", total.Errors)

	return total, nil
}

func shouldSync(opts SyncOptions, source string) bool {
	if len(opts.Sources) == 0 {
		return true
	}
	return slices.Contains(opts.Sources, source)
}

func merge(total, r *SyncResult) {
	total.Created += r.Created
	total.Updated += r.Updated
	total.Linked += r.Linked
	total.Errors += r.Errors
}
