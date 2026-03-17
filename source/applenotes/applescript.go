package applenotes

import (
	"fmt"
	"strings"

	"github.com/73ai/openbotkit/internal/obkmacos"
)

// FetchAll fetches all notes and folders from Apple Notes via the obkmacos helper.
// Returns notes, folders, and a noteAppleID -> Folder map.
func FetchAll() ([]Note, []Folder, map[string]Folder, error) {
	resp, err := obkmacos.FetchNotes()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("fetch notes: %w", err)
	}
	notes, folders, noteToFolder := convertNotesResponse(resp)
	return notes, folders, noteToFolder, nil
}

func convertNotesResponse(resp *obkmacos.NotesResponse) ([]Note, []Folder, map[string]Folder) {
	var notes []Note
	for _, n := range resp.Notes {
		notes = append(notes, Note{
			AppleID:           n.ID,
			Title:             n.Title,
			Body:              n.Body,
			PasswordProtected: n.PasswordProtected,
			CreatedAt:         obkmacos.ParseNoteTime(n.CreatedAt),
			ModifiedAt:        obkmacos.ParseNoteTime(n.ModifiedAt),
		})
	}

	var folders []Folder
	noteToFolder := make(map[string]Folder)
	for _, f := range resp.Folders {
		if isRecentlyDeletedFolder(f.Name) {
			continue
		}
		folder := Folder{
			AppleID:       f.ID,
			Name:          f.Name,
			ParentAppleID: f.ParentID,
			Account:       f.Account,
		}
		folders = append(folders, folder)
		for _, nID := range f.NoteIDs {
			noteToFolder[nID] = folder
		}
	}

	return notes, folders, noteToFolder
}

var recentlyDeletedNames = map[string]bool{
	"recently deleted":         true,
	"récemment supprimées":     true,
	"zuletzt gelöscht":         true,
	"eliminadas recientemente": true,
	"最近削除した項目":                 true,
	"최근 삭제한 항목":                true,
	"最近删除":                     true,
}

func isRecentlyDeletedFolder(name string) bool {
	return recentlyDeletedNames[strings.ToLower(name)]
}

func CheckPermission() error {
	status, err := obkmacos.CheckPermissions()
	if err != nil {
		return err
	}
	if status.Notes != "authorized" {
		return fmt.Errorf("notes permission: %s — grant in System Settings > Privacy & Security > Automation", status.Notes)
	}
	return nil
}
