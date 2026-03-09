package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	ansrc "github.com/priyanshujain/openbotkit/source/applenotes"
	"github.com/priyanshujain/openbotkit/store"
)

type appleNoteItem struct {
	AppleID           string `json:"apple_id"`
	Title             string `json:"title"`
	Body              string `json:"body"`
	Folder            string `json:"folder"`
	FolderID          string `json:"folder_id"`
	Account           string `json:"account"`
	PasswordProtected bool   `json:"password_protected"`
	CreatedAt         string `json:"created_at"`
	ModifiedAt        string `json:"modified_at"`
}

func (s *Server) handleAppleNotesPush(w http.ResponseWriter, r *http.Request) {
	var notes []appleNoteItem
	if err := json.NewDecoder(r.Body).Decode(&notes); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	db, err := store.Open(store.Config{
		Driver: s.cfg.AppleNotes.Storage.Driver,
		DSN:    s.cfg.AppleNotesDataDSN(),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("open db: %v", err))
		return
	}
	defer db.Close()

	if err := ansrc.Migrate(db); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("migrate: %v", err))
		return
	}

	saved := 0
	for _, n := range notes {
		createdAt, _ := time.Parse("2006-01-02T15:04:05Z07:00", n.CreatedAt)
		modifiedAt, _ := time.Parse("2006-01-02T15:04:05Z07:00", n.ModifiedAt)

		note := &ansrc.Note{
			AppleID:           n.AppleID,
			Title:             n.Title,
			Body:              n.Body,
			Folder:            n.Folder,
			FolderID:          n.FolderID,
			Account:           n.Account,
			PasswordProtected: n.PasswordProtected,
			CreatedAt:         createdAt,
			ModifiedAt:        modifiedAt,
		}
		if err := ansrc.SaveNote(db, note); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("save note: %v", err))
			return
		}
		saved++
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"saved": saved})
}
