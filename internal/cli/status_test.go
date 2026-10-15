package cli

import (
	"testing"
	"time"

	wasrc "github.com/73ai/openbotkit/source/whatsapp"
	"github.com/73ai/openbotkit/store"
)

func TestFormatSyncTimeShowsSQLiteUTCInLocalTime(t *testing.T) {
	local := time.Local
	time.Local = time.FixedZone("IST", 5*60*60+30*60)
	t.Cleanup(func() { time.Local = local })

	db, err := store.Open(store.SQLiteConfig(":memory:"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	if err := wasrc.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO whatsapp_messages (account, message_id, chat_jid, synced_at) VALUES ('default', 'm1', '919812345678@s.whatsapp.net', '2026-10-15 06:14:00')`); err != nil {
		t.Fatalf("seed: %v", err)
	}

	last, err := wasrc.LastSyncTime(db)
	if err != nil || last == nil {
		t.Fatalf("last sync: %v, err %v", last, err)
	}
	if got := formatSyncTime(*last); got != "2026-10-15 11:44" {
		t.Fatalf("formatSyncTime = %q, want 2026-10-15 11:44", got)
	}
}
