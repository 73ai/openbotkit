package telegram

import (
	"context"
	"testing"
	"time"

	"github.com/gotd/td/telegram/query/dialogs"
	"github.com/gotd/td/telegram/query/messages"
	"github.com/gotd/td/tg"

	"github.com/73ai/openbotkit/store"
)

func currentIDs(t *testing.T, db *store.DB) map[int64]bool {
	t.Helper()
	chats, err := ListChats(db, 0)
	if err != nil {
		t.Fatalf("list chats: %v", err)
	}
	ids := map[int64]bool{}
	for _, c := range chats {
		ids[c.ChatID] = true
	}
	return ids
}

func TestSyncReconcilesDialogMembership(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	now := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)
	ann, bob := &tg.User{ID: 11, FirstName: "Ann"}, &tg.User{ID: 12, FirstName: "Bob"}
	annDialog, annHist := userDialog(ann, tgMessage(1, now, "hi"))
	bobDialog, bobHist := userDialog(bob, tgMessage(2, now, "yo"))
	f := &fakeFetcher{
		dialogs: []dialogs.Elem{annDialog, bobDialog},
		history: map[int64][]messages.Elem{ChatIDFromUser(11): annHist, ChatIDFromUser(12): bobHist},
	}
	if _, err := Backfill(ctx, f, db, BackfillOptions{}); err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if ids := currentIDs(t, db); !ids[ChatIDFromUser(11)] || !ids[ChatIDFromUser(12)] {
		t.Fatalf("current after first sync = %v", ids)
	}

	f.dialogs = []dialogs.Elem{annDialog}
	if _, err := SyncDialogs(ctx, f, db); err != nil {
		t.Fatalf("sync dialogs: %v", err)
	}
	ids := currentIDs(t, db)
	if !ids[ChatIDFromUser(11)] || ids[ChatIDFromUser(12)] {
		t.Fatalf("current after removal = %v", ids)
	}
	if chat, _ := GetChat(db, ChatIDFromUser(12)); chat == nil {
		t.Fatal("removed chat row should be kept")
	}
	if n, _ := CountMessages(db, ChatIDFromUser(12)); n != 1 {
		t.Fatalf("removed chat history = %d messages, want 1", n)
	}
	if found, _ := FindChats(db, "bob", 0); len(found) != 1 {
		t.Fatalf("FindChats should still see cached peers, got %d", len(found))
	}
	if found, _ := FindCurrentChats(db, "bob", 0); len(found) != 0 {
		t.Fatalf("FindCurrentChats returned removed chat: %v", found)
	}
}

func TestSyncDialogsErrorKeepsMembership(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	d, _ := userDialog(&tg.User{ID: 11, FirstName: "Ann"})
	f := &fakeFetcher{dialogs: []dialogs.Elem{d}}
	if _, err := SyncDialogs(ctx, f, db); err != nil {
		t.Fatalf("sync: %v", err)
	}
	f.dialogsErr = context.DeadlineExceeded
	if _, err := SyncDialogs(ctx, f, db); err == nil {
		t.Fatal("expected error")
	}
	if !currentIDs(t, db)[ChatIDFromUser(11)] {
		t.Fatal("failed sync must not clear membership")
	}
}

func TestSyncSkipsDialogsWithoutPeer(t *testing.T) {
	db := testDB(t)
	d, _ := userDialog(&tg.User{ID: 11, FirstName: "Ann"})
	odd := dialogs.Elem{Dialog: &tg.Dialog{}, Peer: &tg.InputPeerEmpty{}}
	f := &fakeFetcher{dialogs: []dialogs.Elem{odd, d}}
	n, err := SyncDialogs(context.Background(), f, db)
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if n != 1 || !currentIDs(t, db)[ChatIDFromUser(11)] {
		t.Fatalf("count = %d, current = %v", n, currentIDs(t, db))
	}
}

func TestReconcileKeepsChatsActiveDuringSync(t *testing.T) {
	db := testDB(t)
	started := time.Now().UTC().Truncate(time.Second)
	old, fresh := started.Add(-time.Hour), started.Add(time.Second)
	for id, at := range map[int64]time.Time{ChatIDFromUser(11): old, ChatIDFromUser(12): fresh} {
		if err := UpsertChat(db, &Chat{ChatID: id, Type: PeerUser, IsCurrentDialog: true, LastMessageAt: &at}); err != nil {
			t.Fatal(err)
		}
	}
	if err := reconcileDialogs(db, nil, started); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	ids := currentIDs(t, db)
	if ids[ChatIDFromUser(11)] || !ids[ChatIDFromUser(12)] {
		t.Fatalf("current = %v, want only the chat active since sync start", ids)
	}
}

func TestLiveNewMessageMarksChatCurrent(t *testing.T) {
	db := testDB(t)
	d := tg.NewUpdateDispatcher()
	registerUpdateHandlers(d, db, 0, func() {})
	svc := &tg.MessageService{ID: 3, Date: int(time.Now().Unix()), PeerID: &tg.PeerUser{UserID: 11}, Action: &tg.MessageActionContactSignUp{}}
	err := d.Handle(context.Background(), &tg.Updates{
		Updates: []tg.UpdateClass{&tg.UpdateNewMessage{Message: svc}},
		Users:   []tg.UserClass{&tg.User{ID: 11, FirstName: "Ann"}},
	})
	if err != nil {
		t.Fatalf("handle: %v", err)
	}
	if !currentIDs(t, db)[ChatIDFromUser(11)] {
		t.Fatal("new chat from a live update should be current")
	}
}

func TestMigrateAddsDialogMembershipColumn(t *testing.T) {
	db, err := store.Open(store.Config{Driver: "sqlite", DSN: ":memory:"})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE telegram_chats (chat_id INTEGER PRIMARY KEY, type TEXT, title TEXT, username TEXT,
		access_hash INTEGER DEFAULT 0, is_group INTEGER DEFAULT 0, is_channel INTEGER DEFAULT 0, last_message_at DATETIME)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO telegram_chats (chat_id, type, title, username) VALUES (1, 'user', 'Ann', '')`); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := Migrate(db); err != nil {
			t.Fatalf("migrate: %v", err)
		}
	}
	chat, err := GetChat(db, 1)
	if err != nil || chat == nil || chat.Title != "Ann" || chat.IsCurrentDialog {
		t.Fatalf("chat = %+v, err = %v", chat, err)
	}
}

func TestDeleteChatRemovesLocalData(t *testing.T) {
	db := testDB(t)
	keep, gone := ChatIDFromUser(11), ChatIDFromUser(12)
	for _, id := range []int64{keep, gone} {
		if err := UpsertChat(db, &Chat{ChatID: id, Type: PeerUser}); err != nil {
			t.Fatal(err)
		}
		if err := SaveMessage(db, &Message{MessageID: 1, ChatID: id, Timestamp: time.Now()}); err != nil {
			t.Fatal(err)
		}
	}
	if err := DeleteChat(db, gone); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if chat, _ := GetChat(db, gone); chat != nil {
		t.Fatal("chat row not deleted")
	}
	if n, _ := CountMessages(db, gone); n != 0 {
		t.Fatalf("messages left = %d", n)
	}
	if n, _ := CountMessages(db, keep); n != 1 {
		t.Fatalf("other chat lost messages: %d", n)
	}
}
