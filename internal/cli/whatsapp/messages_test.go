package whatsapp

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/pflag"

	wasrc "github.com/73ai/openbotkit/source/whatsapp"
	"github.com/73ai/openbotkit/store"
)

func run(t *testing.T, args ...string) []string {
	t.Helper()
	r, w, _ := os.Pipe()
	stdout := os.Stdout
	os.Stdout = w
	Cmd.SetArgs(args)
	err := Cmd.Execute()
	os.Stdout = stdout
	w.Close()
	out, _ := io.ReadAll(r)
	sub, _, _ := Cmd.Find(args)
	sub.Flags().VisitAll(func(f *pflag.Flag) { f.Value.Set(f.DefValue); f.Changed = false })
	if err != nil {
		t.Fatalf("%v: %v", args, err)
	}
	return strings.Split(strings.TrimSpace(string(out)), "\n")
}

func TestReadCommandsFilterAndLabelAccounts(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", dir)
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
	for i, m := range []wasrc.Message{
		{Account: "default", MessageID: "BAE5F3", Text: "dinner on friday?"},
		{Account: "work", MessageID: "3EB0B7", Text: "sent the invoice"},
	} {
		m.ChatJID, m.SenderJID, m.SenderName = ravi, ravi, "Ravi"
		m.Timestamp = time.Date(2026, 3, 2, 9+i, 0, 0, 0, time.UTC)
		if err := wasrc.SaveMessage(db, &m); err != nil {
			t.Fatalf("seed: %v", err)
		}
		if err := wasrc.UpsertChat(db, m.Account, ravi, "Ravi", false); err != nil {
			t.Fatalf("seed chat: %v", err)
		}
	}

	all := run(t, "messages", "list")
	if len(all) != 3 || !strings.HasPrefix(all[0], "ACCOUNT") || !strings.HasPrefix(all[1], "work ") || !strings.HasPrefix(all[2], "default ") {
		t.Fatalf("messages list:\n%s", strings.Join(all, "\n"))
	}
	work := run(t, "messages", "search", "invoice", "--account", "work")
	if len(work) != 2 || !strings.HasPrefix(work[1], "work ") || !strings.Contains(work[1], "sent the invoice") {
		t.Fatalf("messages search --account work:\n%s", strings.Join(work, "\n"))
	}
	if none := run(t, "messages", "search", "invoice", "--account", "default"); len(none) != 1 || none[0] != "No messages found." {
		t.Fatalf("messages search --account default:\n%s", strings.Join(none, "\n"))
	}
	chats := run(t, "chats", "list", "--account", "default")
	if len(chats) != 2 || !strings.HasPrefix(chats[1], "default ") {
		t.Fatalf("chats list --account default:\n%s", strings.Join(chats, "\n"))
	}
}
