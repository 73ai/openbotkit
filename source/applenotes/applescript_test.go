package applenotes

import (
	"testing"

	"github.com/73ai/openbotkit/internal/obkmacos"
)

func TestIsRecentlyDeletedFolder(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"Recently Deleted", true},
		{"recently deleted", true},
		{"Notes", false},
		{"Work", false},
		{"Récemment supprimées", true},
		{"Zuletzt gelöscht", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRecentlyDeletedFolder(tt.name)
			if got != tt.want {
				t.Errorf("isRecentlyDeletedFolder(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestConvertNotesResponse(t *testing.T) {
	resp := &obkmacos.NotesResponse{
		Notes: []obkmacos.Note{
			{
				ID:                "note-1",
				Title:             "My Note",
				Body:              "hello",
				PasswordProtected: false,
				CreatedAt:         "2026-01-15T10:30:00Z",
				ModifiedAt:        "2026-03-10T14:22:00Z",
			},
			{
				ID:                "note-2",
				Title:             "Secret",
				Body:              "",
				PasswordProtected: true,
				CreatedAt:         "2026-02-01T08:00:00Z",
				ModifiedAt:        "2026-02-01T08:00:00Z",
			},
		},
		Folders: []obkmacos.Folder{
			{
				ID:      "folder-1",
				Name:    "Work",
				Account: "iCloud",
				NoteIDs: []string{"note-1"},
			},
			{
				ID:      "folder-deleted",
				Name:    "Recently Deleted",
				Account: "iCloud",
				NoteIDs: []string{"note-2"},
			},
		},
	}

	notes, folders, noteToFolder := convertNotesResponse(resp)

	if len(notes) != 2 {
		t.Fatalf("expected 2 notes, got %d", len(notes))
	}
	if notes[0].AppleID != "note-1" || notes[0].Title != "My Note" {
		t.Errorf("note[0]: got id=%q title=%q", notes[0].AppleID, notes[0].Title)
	}
	if notes[0].CreatedAt.IsZero() {
		t.Error("note[0] CreatedAt should not be zero")
	}
	if notes[1].PasswordProtected != true {
		t.Error("note[1] should be password protected")
	}

	// "Recently Deleted" folder should be filtered out
	if len(folders) != 1 {
		t.Fatalf("expected 1 folder (Recently Deleted filtered), got %d", len(folders))
	}
	if folders[0].Name != "Work" {
		t.Errorf("folder[0]: got name=%q", folders[0].Name)
	}

	// note-1 should map to Work folder
	if f, ok := noteToFolder["note-1"]; !ok || f.Name != "Work" {
		t.Errorf("noteToFolder[note-1]: got %v", noteToFolder["note-1"])
	}
	// note-2 was in Recently Deleted, should not be in map
	if _, ok := noteToFolder["note-2"]; ok {
		t.Error("noteToFolder should not contain note-2 (was in Recently Deleted)")
	}
}

func TestConvertNotesResponse_Empty(t *testing.T) {
	resp := &obkmacos.NotesResponse{}
	notes, folders, noteToFolder := convertNotesResponse(resp)

	if len(notes) != 0 {
		t.Errorf("expected 0 notes, got %d", len(notes))
	}
	if len(folders) != 0 {
		t.Errorf("expected 0 folders, got %d", len(folders))
	}
	if len(noteToFolder) != 0 {
		t.Errorf("expected empty noteToFolder, got %d entries", len(noteToFolder))
	}
}
