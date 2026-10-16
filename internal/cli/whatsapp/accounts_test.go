package whatsapp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/73ai/openbotkit/config"
	wasrc "github.com/73ai/openbotkit/source/whatsapp"
	"github.com/73ai/openbotkit/store"
)

func TestAccountsListShowsPerAccountSync(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", dir)
	cfg := config.Default()
	cfg.SetWhatsAppAccount("work", config.WhatsAppAccount{Role: "source"})
	cfg.SetWhatsAppAccount("assistant", config.WhatsAppAccount{Role: "channel", OwnerJID: "919876543210@s.whatsapp.net"})
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}
	for _, label := range []string{"default", "work"} {
		if err := config.LinkWhatsAppAccount(label); err != nil {
			t.Fatalf("link %s: %v", label, err)
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, "whatsapp"), 0700); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(store.SQLiteConfig(filepath.Join(dir, "whatsapp", "data.db")))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	if err := wasrc.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	ravi := "919812345678@s.whatsapp.net"
	for _, m := range []struct{ account, id, syncedAt string }{
		{"default", "BAE5F3", "2026-10-15 21:40:00"},
		{"default", "BAE5F4", "2026-10-14 08:00:00"},
		{"work", "3EB0B7", "2026-09-30 12:15:00"},
	} {
		if err := wasrc.SaveMessage(db, &wasrc.Message{Account: m.account, MessageID: m.id, ChatJID: ravi, Timestamp: time.Now()}); err != nil {
			t.Fatalf("seed: %v", err)
		}
		if _, err := db.Exec("UPDATE whatsapp_messages SET synced_at = ? WHERE message_id = ?", m.syncedAt, m.id); err != nil {
			t.Fatalf("set synced_at: %v", err)
		}
	}

	local := func(y int, mo time.Month, d, h, mi int) string {
		return time.Date(y, mo, d, h, mi, 0, 0, time.UTC).Local().Format("2006-01-02 15:04")
	}
	want := [][]string{
		{"LABEL", "ROLE", "LINKED", "MESSAGES", "LAST", "SYNC", "OWNER"},
		{"assistant", "channel", "no", "0", "never", "919876543210@s.whatsapp.net"},
		append([]string{"default", "source", "yes", "2"}, append(strings.Fields(local(2026, 10, 15, 21, 40)), "-")...),
		append([]string{"work", "source", "yes", "1"}, append(strings.Fields(local(2026, 9, 30, 12, 15)), "-")...),
	}
	got := run(t, "accounts", "list")
	if len(got) != len(want) {
		t.Fatalf("accounts list:\n%s", strings.Join(got, "\n"))
	}
	for i, line := range got {
		if strings.Join(strings.Fields(line), " ") != strings.Join(want[i], " ") {
			t.Errorf("row %d = %q, want %q", i, line, strings.Join(want[i], " "))
		}
	}
}
