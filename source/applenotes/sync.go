package applenotes

import (
	"fmt"
	"log/slog"

	"github.com/73ai/openbotkit/store"
)

func Sync(db *store.DB, opts SyncOptions) (*SyncResult, error) {
	if err := Migrate(db); err != nil {
		return nil, fmt.Errorf("migrate schema: %w", err)
	}

	slog.Info("applenotes: fetching notes and folders")
	notes, folders, noteToFolder, err := FetchAll()
	if err != nil {
		return nil, fmt.Errorf("fetch notes: %w", err)
	}
	slog.Info("applenotes: found", "notes", len(notes), "folders", len(folders))

	for i := range folders {
		if err := SaveFolder(db, &folders[i]); err != nil {
			slog.Error("applenotes: save folder", "folder", folders[i].Name, "error", err)
		}
	}

	result := &SyncResult{}
	for i := range notes {
		n := &notes[i]

		// Attach folder info from the folder map
		if f, ok := noteToFolder[n.AppleID]; ok {
			n.Folder = f.Name
			n.FolderID = f.AppleID
			n.Account = f.Account
		}

		// Skip password-protected notes body (metadata is still saved)
		if n.PasswordProtected {
			n.Body = ""
		}

		if err := SaveNote(db, n); err != nil {
			slog.Error("applenotes: save note", "title", n.Title, "error", err)
			result.Errors++
			continue
		}
		result.Synced++
	}

	return result, nil
}
