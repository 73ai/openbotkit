package jobs

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/73ai/openbotkit/channel"
	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/service/hooks"
	gmailsrc "github.com/73ai/openbotkit/source/gmail"
	"github.com/73ai/openbotkit/store"
)

func TestTruncate(t *testing.T) {
	cases := []struct {
		input string
		n     int
		want  string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hello..."},
		{"", 5, ""},
		{"ab", 2, "ab"},
		{"abc", 2, "ab..."},
	}
	for _, tc := range cases {
		got := truncate(tc.input, tc.n)
		if got != tc.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tc.input, tc.n, got, tc.want)
		}
	}
}

func TestEventHookArgs_Kind(t *testing.T) {
	args := EventHookArgs{}
	if args.Kind() != "event_hook" {
		t.Errorf("Kind() = %q, want event_hook", args.Kind())
	}
}

func setupHookTest(t *testing.T) (*store.DB, *store.DB, *channel.Registry) {
	t.Helper()
	dir := t.TempDir()

	hooksDB, err := store.Open(store.SQLiteConfig(filepath.Join(dir, "hooks.db")))
	if err != nil {
		t.Fatalf("open hooks db: %v", err)
	}
	t.Cleanup(func() { hooksDB.Close() })
	hooks.Migrate(hooksDB)

	gmailDB, err := store.Open(store.SQLiteConfig(filepath.Join(dir, "gmail.db")))
	if err != nil {
		t.Fatalf("open gmail db: %v", err)
	}
	t.Cleanup(func() { gmailDB.Close() })
	gmailsrc.Migrate(gmailDB)

	chanReg := channel.NewRegistry()
	return hooksDB, gmailDB, chanReg
}

func makeJob(hookID int64, ids []int64) *river.Job[EventHookArgs] {
	return &river.Job[EventHookArgs]{
		JobRow: &rivertype.JobRow{},
		Args:   EventHookArgs{HookID: hookID, ItemIDs: ids},
	}
}

func TestWork_HookNotFound(t *testing.T) {
	hooksDB, _, chanReg := setupHookTest(t)

	w := &EventHookWorker{
		Cfg:     config.Default(),
		ChanReg: chanReg,
		HooksDB: hooksDB,
	}

	err := w.Work(context.Background(), makeJob(999, []int64{1}))
	if err == nil {
		t.Error("expected error for missing hook")
	}
}

func TestWork_EmptyEmailIDs(t *testing.T) {
	hooksDB, _, chanReg := setupHookTest(t)

	hookID, _ := hooks.Create(hooksDB, &hooks.EventHook{
		EventType: "gmail_sync",
		Prompt:    "classify",
		Channel:   "telegram",
		ModelTier: "nano",
	})

	w := &EventHookWorker{
		Cfg:     config.Default(),
		ChanReg: chanReg,
		HooksDB: hooksDB,
	}

	// Empty IDs → should return nil immediately.
	err := w.Work(context.Background(), makeJob(hookID, []int64{}))
	if err != nil {
		t.Errorf("expected nil for empty IDs, got: %v", err)
	}
}

func TestWork_NoModelsConfigured(t *testing.T) {
	dir := t.TempDir()
	hooksDB, err := store.Open(store.SQLiteConfig(filepath.Join(dir, "hooks.db")))
	if err != nil {
		t.Fatalf("open hooks db: %v", err)
	}
	defer hooksDB.Close()
	hooks.Migrate(hooksDB)

	gmailDBPath := filepath.Join(dir, "gmail.db")
	gmailDB, _ := store.Open(store.SQLiteConfig(gmailDBPath))
	gmailsrc.Migrate(gmailDB)
	gmailsrc.SaveEmail(gmailDB, &gmailsrc.Email{
		MessageID: "msg1",
		Account:   "test@example.com",
		From:      "alice@example.com",
		Subject:   "Test",
		Body:      "Hello",
		Date:      time.Now(),
	})
	gmailDB.Close()

	hookID, _ := hooks.Create(hooksDB, &hooks.EventHook{
		EventType: "gmail_sync",
		Prompt:    "classify",
		Channel:   "telegram",
		ModelTier: "nano",
	})

	cfg := config.Default()
	cfg.Gmail.Storage.Driver = "sqlite"
	cfg.Gmail.Storage.DSN = gmailDBPath
	cfg.Models = nil // no models configured

	chanReg := channel.NewRegistry()

	w := &EventHookWorker{
		Cfg:     cfg,
		ChanReg: chanReg,
		HooksDB: hooksDB,
	}

	err = w.Work(context.Background(), makeJob(hookID, []int64{1}))
	if err == nil {
		t.Error("expected error for nil models")
	}
}
