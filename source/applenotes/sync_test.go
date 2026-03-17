//go:build darwin && integration

package applenotes

import (
	"path/filepath"
	"testing"

	"github.com/73ai/openbotkit/store"
)

func TestCheckPermission_Integration(t *testing.T) {
	if err := CheckPermission(); err != nil {
		t.Logf("permission check failed (may need to grant access): %v", err)
	}
}

func TestFetchAll_Integration(t *testing.T) {
	notes, folders, noteToFolder, err := FetchAll()
	if err != nil {
		t.Fatalf("FetchAll: %v", err)
	}

	t.Logf("fetched %d notes, %d folders, %d note-to-folder mappings",
		len(notes), len(folders), len(noteToFolder))

	for i, n := range notes {
		if i >= 5 {
			break
		}
		bodyPreview := n.Body
		if len(bodyPreview) > 80 {
			bodyPreview = bodyPreview[:77] + "..."
		}
		t.Logf("  note[%d] title=%q folder=%q protected=%v body=%q",
			i, n.Title, n.Folder, n.PasswordProtected, bodyPreview)
	}

	for i, f := range folders {
		if i >= 5 {
			break
		}
		t.Logf("  folder[%d] name=%q account=%q parent=%q",
			i, f.Name, f.Account, f.ParentAppleID)
	}
}

func TestSync_Integration(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := store.Open(store.Config{Driver: "sqlite", DSN: dbPath})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	result, err := Sync(db, SyncOptions{})
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}

	t.Logf("sync result: synced=%d skipped=%d errors=%d",
		result.Synced, result.Skipped, result.Errors)

	count, _ := CountNotes(db)
	t.Logf("total notes in DB: %d", count)

	if count == 0 && result.Errors == 0 {
		t.Log("warning: no notes synced (Notes app may be empty or permissions not granted)")
	}

	notes, _ := ListNotes(db, ListOptions{Limit: 5})
	for _, n := range notes {
		t.Logf("  %q in %q (modified %s)", n.Title, n.Folder, n.ModifiedAt.Format("2006-01-02"))
	}
}
