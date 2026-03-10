package imessage

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/priyanshujain/openbotkit/store"
)

func openTestDB(t *testing.T) *store.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := store.Open(store.Config{Driver: "sqlite", DSN: dbPath})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestMigrate(t *testing.T) {
	db := openTestDB(t)

	if err := Migrate(db); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
}

func TestSaveAndGetMessage(t *testing.T) {
	db := openTestDB(t)

	now := time.Now().Truncate(time.Second)
	msg := &Message{
		GUID:            "msg-001",
		AppleROWID:      100,
		Text:            "Hello world",
		ChatGUID:        "iMessage;-;+1234567890",
		SenderID:        "+1234567890",
		SenderService:   "iMessage",
		IsFromMe:        false,
		IsRead:          true,
		Date:            now.Add(-time.Hour),
		DateRead:        now,
		ChatDisplayName: "John",
	}

	if err := SaveMessage(db, msg); err != nil {
		t.Fatalf("save message: %v", err)
	}

	got, err := GetMessage(db, msg.GUID)
	if err != nil {
		t.Fatalf("get message: %v", err)
	}

	if got.Text != "Hello world" {
		t.Errorf("text = %q, want %q", got.Text, "Hello world")
	}
	if got.ChatGUID != "iMessage;-;+1234567890" {
		t.Errorf("chat_guid = %q, want %q", got.ChatGUID, "iMessage;-;+1234567890")
	}
	if got.IsFromMe {
		t.Error("is_from_me should be false")
	}
	if !got.IsRead {
		t.Error("is_read should be true")
	}
	if got.AppleROWID != 100 {
		t.Errorf("apple_rowid = %d, want 100", got.AppleROWID)
	}
}

func TestSaveMessageUpsert(t *testing.T) {
	db := openTestDB(t)

	msg := &Message{
		GUID:       "msg-002",
		AppleROWID: 200,
		Text:       "original",
	}
	if err := SaveMessage(db, msg); err != nil {
		t.Fatalf("save: %v", err)
	}

	msg.Text = "updated"
	if err := SaveMessage(db, msg); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	got, err := GetMessage(db, msg.GUID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Text != "updated" {
		t.Errorf("text = %q, want %q", got.Text, "updated")
	}
}

func TestListMessages(t *testing.T) {
	db := openTestDB(t)

	now := time.Now().Truncate(time.Second)
	for i, text := range []string{"Msg A", "Msg B", "Msg C"} {
		m := &Message{
			GUID:       "list-" + text,
			AppleROWID: int64(300 + i),
			Text:       text,
			Date:       now.Add(time.Duration(i) * time.Minute),
		}
		if err := SaveMessage(db, m); err != nil {
			t.Fatalf("save %q: %v", text, err)
		}
	}

	msgs, err := ListMessages(db, ListOptions{Limit: 10})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(msgs) != 3 {
		t.Fatalf("count = %d, want 3", len(msgs))
	}
	if msgs[0].Text != "Msg C" {
		t.Errorf("first = %q, want %q", msgs[0].Text, "Msg C")
	}
}

func TestListMessagesByChatGUID(t *testing.T) {
	db := openTestDB(t)

	for _, m := range []Message{
		{GUID: "chat-1", AppleROWID: 400, Text: "In chat A", ChatGUID: "chatA"},
		{GUID: "chat-2", AppleROWID: 401, Text: "In chat B", ChatGUID: "chatB"},
		{GUID: "chat-3", AppleROWID: 402, Text: "Also chat A", ChatGUID: "chatA"},
	} {
		if err := SaveMessage(db, &m); err != nil {
			t.Fatalf("save: %v", err)
		}
	}

	msgs, err := ListMessages(db, ListOptions{ChatGUID: "chatA", Limit: 50})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("count = %d, want 2", len(msgs))
	}
}

func TestSearchMessages(t *testing.T) {
	db := openTestDB(t)

	for _, m := range []Message{
		{GUID: "s1", AppleROWID: 500, Text: "meeting tomorrow"},
		{GUID: "s2", AppleROWID: 501, Text: "buy groceries"},
		{GUID: "s3", AppleROWID: 502, Text: "project meeting notes"},
	} {
		if err := SaveMessage(db, &m); err != nil {
			t.Fatalf("save: %v", err)
		}
	}

	msgs, err := SearchMessages(db, "meeting", 50)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(msgs) != 2 {
		t.Errorf("count = %d, want 2", len(msgs))
	}
}

func TestCountMessages(t *testing.T) {
	db := openTestDB(t)

	count, err := CountMessages(db)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}

	SaveMessage(db, &Message{GUID: "c1", AppleROWID: 600, Text: "One"})
	SaveMessage(db, &Message{GUID: "c2", AppleROWID: 601, Text: "Two"})

	count, err = CountMessages(db)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestLastSyncTime(t *testing.T) {
	db := openTestDB(t)

	last, err := LastSyncTime(db)
	if err != nil {
		t.Fatalf("last sync: %v", err)
	}
	if last != nil {
		t.Errorf("expected nil, got %v", last)
	}

	SaveMessage(db, &Message{GUID: "lt1", AppleROWID: 700, Text: "First"})
	last, err = LastSyncTime(db)
	if err != nil {
		t.Fatalf("last sync: %v", err)
	}
	if last == nil {
		t.Fatal("expected non-nil time")
	}
}

func TestMaxAppleROWID(t *testing.T) {
	db := openTestDB(t)

	rowid, err := MaxAppleROWID(db)
	if err != nil {
		t.Fatalf("max rowid: %v", err)
	}
	if rowid != 0 {
		t.Errorf("rowid = %d, want 0", rowid)
	}

	SaveMessage(db, &Message{GUID: "r1", AppleROWID: 50, Text: "Low"})
	SaveMessage(db, &Message{GUID: "r2", AppleROWID: 999, Text: "High"})
	SaveMessage(db, &Message{GUID: "r3", AppleROWID: 100, Text: "Mid"})

	rowid, err = MaxAppleROWID(db)
	if err != nil {
		t.Fatalf("max rowid: %v", err)
	}
	if rowid != 999 {
		t.Errorf("rowid = %d, want 999", rowid)
	}
}

func TestSaveAndListChats(t *testing.T) {
	db := openTestDB(t)

	now := time.Now().Truncate(time.Second)
	chat := &Chat{
		GUID:            "iMessage;-;chat123",
		DisplayName:     "Group Chat",
		ServiceName:     "iMessage",
		Participants:    []string{"+1234567890", "+0987654321"},
		IsGroup:         true,
		LastMessageDate: now,
	}

	if err := SaveChat(db, chat); err != nil {
		t.Fatalf("save chat: %v", err)
	}

	chats, err := ListChats(db, ListOptions{Limit: 10})
	if err != nil {
		t.Fatalf("list chats: %v", err)
	}
	if len(chats) != 1 {
		t.Fatalf("count = %d, want 1", len(chats))
	}
	if chats[0].DisplayName != "Group Chat" {
		t.Errorf("display_name = %q, want %q", chats[0].DisplayName, "Group Chat")
	}
	if !chats[0].IsGroup {
		t.Error("is_group should be true")
	}
	if len(chats[0].Participants) != 2 {
		t.Errorf("participants count = %d, want 2", len(chats[0].Participants))
	}
}

func TestSaveHandle(t *testing.T) {
	db := openTestDB(t)

	h := &Handle{ID: "+1234567890", Service: "iMessage"}
	if err := SaveHandle(db, h); err != nil {
		t.Fatalf("save handle: %v", err)
	}

	// Upsert with different service
	h.Service = "SMS"
	if err := SaveHandle(db, h); err != nil {
		t.Fatalf("upsert handle: %v", err)
	}

	var service string
	err := db.QueryRow(db.Rebind("SELECT service FROM imessage_handles WHERE handle_id = ?"), h.ID).Scan(&service)
	if err != nil {
		t.Fatalf("query handle: %v", err)
	}
	if service != "SMS" {
		t.Errorf("service = %q, want %q", service, "SMS")
	}
}

func TestListMessagesModifiedSince(t *testing.T) {
	db := openTestDB(t)

	// Use a time well in the past as the "since" marker.
	before := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	for _, m := range []Message{
		{GUID: "mod-1", AppleROWID: 800, Text: "Old message"},
		{GUID: "mod-2", AppleROWID: 801, Text: "New message"},
	} {
		if err := SaveMessage(db, &m); err != nil {
			t.Fatalf("save: %v", err)
		}
	}

	// synced_at is CURRENT_TIMESTAMP (UTC), which is after 2020.
	msgs, err := ListMessagesModifiedSince(db, before)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(msgs) != 2 {
		t.Errorf("count = %d, want 2", len(msgs))
	}

	// A future time should return none.
	future := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	msgs, err = ListMessagesModifiedSince(db, future)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("count = %d, want 0", len(msgs))
	}
}
