package whatsapp

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.mau.fi/whatsmeow/proto/waCommon"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/proto/waHistorySync"
	"go.mau.fi/whatsmeow/proto/waWeb"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"

	"github.com/73ai/openbotkit/store"
)

func migratedDB(t *testing.T) *store.DB {
	t.Helper()
	db := testDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestAccountsKeepSeparateMessagesChatsAndContacts(t *testing.T) {
	db := migratedDB(t)
	group := "team@g.us"
	now := time.Now()

	for _, account := range []string{"default", "work"} {
		if err := SaveMessage(db, &Message{Account: account, MessageID: "shared", ChatJID: group, Text: "hello from " + account, Timestamp: now}); err != nil {
			t.Fatalf("save %s: %v", account, err)
		}
		if err := UpsertChat(db, account, group, "Team "+account, true); err != nil {
			t.Fatalf("upsert chat %s: %v", account, err)
		}
		if err := SaveContact(db, &Contact{Account: account, JID: "bob@s.whatsapp.net", FullName: "Bob " + account}); err != nil {
			t.Fatalf("save contact %s: %v", account, err)
		}
	}
	if err := SaveMessage(db, &Message{Account: "work", MessageID: "only-work", ChatJID: group, Text: "standup notes", Timestamp: now}); err != nil {
		t.Fatalf("save work-only: %v", err)
	}

	all, err := ListMessages(db, ListOptions{})
	if err != nil || len(all) != 3 {
		t.Fatalf("all messages = %d, err %v", len(all), err)
	}
	work, err := ListMessages(db, ListOptions{Account: "work"})
	if err != nil || len(work) != 2 {
		t.Fatalf("work messages = %d, err %v", len(work), err)
	}
	for _, m := range work {
		if m.Account != "work" {
			t.Fatalf("work filter returned account %q", m.Account)
		}
	}

	found, err := SearchMessages(db, "default", "hello", 10)
	if err != nil || len(found) != 1 || found[0].Text != "hello from default" {
		t.Fatalf("search default = %+v, err %v", found, err)
	}
	found, err = SearchMessages(db, "", "hello", 10)
	if err != nil || len(found) != 2 {
		t.Fatalf("search all = %d, err %v", len(found), err)
	}

	chats, err := ListChats(db, "work")
	if err != nil || len(chats) != 1 || chats[0].Name != "Team work" || chats[0].Account != "work" {
		t.Fatalf("work chats = %+v, err %v", chats, err)
	}
	chats, err = ListChats(db, "")
	if err != nil || len(chats) != 2 {
		t.Fatalf("all chats = %d, err %v", len(chats), err)
	}

	contact, err := GetContact(db, "work", "bob@s.whatsapp.net")
	if err != nil || contact == nil || contact.FullName != "Bob work" {
		t.Fatalf("work contact = %+v, err %v", contact, err)
	}
	contacts, err := ListContacts(db, "default", "bob", 10)
	if err != nil || len(contacts) != 1 || contacts[0].FullName != "Bob default" {
		t.Fatalf("default contacts = %+v, err %v", contacts, err)
	}

	exists, err := MessageExists(db, "default", "only-work", group)
	if err != nil || exists {
		t.Fatalf("work-only message visible in default: %v, err %v", exists, err)
	}
}

func TestAccountScopedCallsRequireAccount(t *testing.T) {
	db := migratedDB(t)
	ctx := context.Background()
	chat := "919812345678@s.whatsapp.net"

	calls := map[string]error{
		"SaveMessage": SaveMessage(db, &Message{MessageID: "m1", ChatJID: chat, Text: "hi", Timestamp: time.Now()}),
		"UpsertChat":  UpsertChat(db, "", chat, "Ravi", false),
		"SaveContact": SaveContact(db, &Contact{JID: chat, FullName: "Ravi"}),
	}
	_, calls["SendText"] = SendText(ctx, nil, db, SendInput{ChatJID: chat, Text: "hi"})
	_, calls["Sync"] = Sync(ctx, nil, db, SyncOptions{Follow: true})
	_, calls["MessageExists"] = MessageExists(db, "", "m1", chat)
	_, calls["GetContact"] = GetContact(db, "", chat)
	for name, err := range calls {
		if !errors.Is(err, errNoAccount) {
			t.Errorf("%s without account: err = %v, want errNoAccount", name, err)
		}
	}

	for _, table := range []string{"whatsapp_messages", "whatsapp_chats", "whatsapp_contacts"} {
		var n int
		if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil || n != 0 {
			t.Fatalf("%s rows = %d, err %v", table, n, err)
		}
	}
}

func TestSaveHistorySyncTagsAccount(t *testing.T) {
	db := migratedDB(t)
	evt := &events.HistorySync{Data: &waHistorySync.HistorySync{
		Conversations: []*waHistorySync.Conversation{{
			ID:          proto.String("alice@s.whatsapp.net"),
			DisplayName: proto.String("Alice"),
			Messages: []*waHistorySync.HistorySyncMsg{{
				Message: &waWeb.WebMessageInfo{
					Key: &waCommon.MessageKey{
						RemoteJID: proto.String("alice@s.whatsapp.net"),
						ID:        proto.String("h1"),
					},
					Message:          &waE2E.Message{Conversation: proto.String("from history")},
					MessageTimestamp: proto.Uint64(uint64(time.Now().Unix())),
				},
			}},
		}},
	}}

	saved, failed := saveHistorySync(db, "work", evt)
	if saved != 1 || failed != 0 {
		t.Fatalf("saved %d failed %d", saved, failed)
	}
	msgs, err := ListMessages(db, ListOptions{Account: "work"})
	if err != nil || len(msgs) != 1 || msgs[0].SenderName != "Alice" {
		t.Fatalf("work history = %+v, err %v", msgs, err)
	}
	chats, err := ListChats(db, "work")
	if err != nil || len(chats) != 1 || chats[0].Name != "Alice" {
		t.Fatalf("work chats = %+v, err %v", chats, err)
	}
	if msgs, _ := ListMessages(db, ListOptions{Account: DefaultAccount}); len(msgs) != 0 {
		t.Fatalf("history leaked into default: %d", len(msgs))
	}
}

func TestSaveHistorySyncCountsChatFailures(t *testing.T) {
	db := migratedDB(t)
	if _, err := db.Exec("DROP TABLE whatsapp_chats"); err != nil {
		t.Fatalf("drop chats: %v", err)
	}
	evt := &events.HistorySync{Data: &waHistorySync.HistorySync{
		Conversations: []*waHistorySync.Conversation{{
			ID:          proto.String("alice@s.whatsapp.net"),
			DisplayName: proto.String("Alice"),
			Messages: []*waHistorySync.HistorySyncMsg{{
				Message: &waWeb.WebMessageInfo{
					Key:              &waCommon.MessageKey{RemoteJID: proto.String("alice@s.whatsapp.net"), ID: proto.String("h1")},
					Message:          &waE2E.Message{Conversation: proto.String("from history")},
					MessageTimestamp: proto.Uint64(uint64(time.Now().Unix())),
				},
			}},
		}},
	}}

	saved, failed := saveHistorySync(db, "work", evt)
	if saved != 1 || failed != 1 {
		t.Fatalf("saved %d failed %d, want the message saved and the chat counted as failed", saved, failed)
	}
}

func TestSaveLiveMessageTagsAccount(t *testing.T) {
	db := migratedDB(t)
	ravi := types.NewJID("919812345678", types.DefaultUserServer)
	evt := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{Chat: ravi, Sender: ravi},
			ID:            "3EB0C4F1A2",
			PushName:      "Ravi",
			Timestamp:     time.Date(2026, 3, 4, 10, 30, 0, 0, time.UTC),
		},
		Message: &waE2E.Message{Conversation: proto.String("are we still on for lunch?")},
	}

	msg, err := saveLiveMessage(db, "work", evt)
	if err != nil || msg == nil {
		t.Fatalf("save live message: %+v, err %v", msg, err)
	}
	stored, err := ListMessages(db, ListOptions{Account: "work"})
	if err != nil || len(stored) != 1 || stored[0].Text != "are we still on for lunch?" || stored[0].ChatJID != ravi.String() {
		t.Fatalf("work messages = %+v, err %v", stored, err)
	}
	chats, err := ListChats(db, "work")
	if err != nil || len(chats) != 1 || chats[0].Name != "Ravi" {
		t.Fatalf("work chats = %+v, err %v", chats, err)
	}
	if others, _ := ListMessages(db, ListOptions{Account: DefaultAccount}); len(others) != 0 {
		t.Fatalf("live message leaked into default: %+v", others)
	}
}

func TestStatusCoversAllAccountsWithoutCreatingSessions(t *testing.T) {
	db := migratedDB(t)
	for _, account := range []string{"default", "work"} {
		if err := SaveMessage(db, &Message{Account: account, MessageID: "m-" + account, ChatJID: "919812345678@s.whatsapp.net", Timestamp: time.Now()}); err != nil {
			t.Fatalf("save %s: %v", account, err)
		}
	}
	dir := t.TempDir()
	sessions := []string{filepath.Join(dir, "session.db"), filepath.Join(dir, "work", "session.db")}

	st, err := New(Config{SessionDBPaths: sessions}).Status(context.Background(), db)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if st.Connected || len(st.Accounts) != 0 || st.ItemCount != 2 {
		t.Fatalf("status = %+v, want disconnected with 2 messages", st)
	}
	for _, p := range sessions {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatalf("status created session %s", p)
		}
	}
}
