package history

import (
	"fmt"
	"testing"
	"time"
)

func TestParseTimestamp(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"sqlite format", "2026-03-13 10:30:00", true},
		{"iso8601 utc", "2026-03-13T10:30:00Z", true},
		{"rfc3339", "2026-03-13T10:30:00+05:30", true},
		{"invalid", "not-a-date", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTimestamp(tt.input)
			if tt.want && got == nil {
				t.Errorf("parseTimestamp(%q) = nil, want non-nil", tt.input)
			}
			if !tt.want && got != nil {
				t.Errorf("parseTimestamp(%q) = %v, want nil", tt.input, got)
			}
		})
	}
}

func TestUpsertConversation(t *testing.T) {
	db := testDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	id1, err := UpsertConversation(db, "session-001", "/tmp/project")
	if err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	if id1 == 0 {
		t.Fatal("expected non-zero id")
	}

	// Upsert same session should return same id.
	id2, err := UpsertConversation(db, "session-001", "/tmp/project")
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	if id1 != id2 {
		t.Fatalf("expected same id %d, got %d", id1, id2)
	}

	// Different session should get different id.
	id3, err := UpsertConversation(db, "session-002", "/tmp/other")
	if err != nil {
		t.Fatalf("third upsert: %v", err)
	}
	if id3 == id1 {
		t.Fatal("expected different id for different session")
	}
}

func TestSaveMessage(t *testing.T) {
	db := testDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	convID, err := UpsertConversation(db, "session-001", "/tmp")
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}

	if err := SaveMessage(db, convID, "user", "hello"); err != nil {
		t.Fatalf("save user message: %v", err)
	}
	if err := SaveMessage(db, convID, "assistant", "hi there"); err != nil {
		t.Fatalf("save assistant message: %v", err)
	}

	count, err := MessageCountForSession(db, "session-001")
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 messages, got %d", count)
	}
}

func TestCountConversations(t *testing.T) {
	db := testDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	count, err := CountConversations(db)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0, got %d", count)
	}

	UpsertConversation(db, "s1", "/tmp")
	UpsertConversation(db, "s2", "/tmp")

	count, err = CountConversations(db)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2, got %d", count)
	}
}

func TestLastCaptureTime(t *testing.T) {
	db := testDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	ts, err := LastCaptureTime(db)
	if err != nil {
		t.Fatalf("last capture: %v", err)
	}
	if ts != nil {
		t.Fatalf("expected nil, got %v", ts)
	}

	UpsertConversation(db, "s1", "/tmp")

	ts, err = LastCaptureTime(db)
	if err != nil {
		t.Fatalf("last capture: %v", err)
	}
	if ts == nil {
		t.Fatal("expected non-nil timestamp after upsert")
	}
}

func TestMessageCountForSession(t *testing.T) {
	db := testDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	count, err := MessageCountForSession(db, "nonexistent")
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0, got %d", count)
	}
}

func TestLoadSessionMessages_RoundTrip(t *testing.T) {
	db := testDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	convID, _ := UpsertConversation(db, "tg-msg", "telegram")
	SaveMessage(db, convID, "user", "hello")
	SaveMessage(db, convID, "assistant", "hi there")
	SaveMessage(db, convID, "user", "bye")

	msgs, err := LoadSessionMessages(db, "tg-msg", 100)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}
	if msgs[0].Role != "user" || msgs[0].Content != "hello" {
		t.Errorf("msg[0] = %q/%q", msgs[0].Role, msgs[0].Content)
	}
	if msgs[1].Role != "assistant" || msgs[1].Content != "hi there" {
		t.Errorf("msg[1] = %q/%q", msgs[1].Role, msgs[1].Content)
	}
	if msgs[2].Role != "user" || msgs[2].Content != "bye" {
		t.Errorf("msg[2] = %q/%q", msgs[2].Role, msgs[2].Content)
	}
}

func TestLoadSessionMessages_EmptySession(t *testing.T) {
	db := testDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	msgs, err := LoadSessionMessages(db, "nonexistent", 100)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(msgs) != 0 {
		t.Fatalf("expected 0 messages, got %d", len(msgs))
	}
}

func TestLoadSessionMessages_Limit(t *testing.T) {
	db := testDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	convID, _ := UpsertConversation(db, "tg-limit", "telegram")
	for i := range 10 {
		SaveMessage(db, convID, "user", fmt.Sprintf("msg %d", i))
	}

	msgs, err := LoadSessionMessages(db, "tg-limit", 5)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(msgs) != 5 {
		t.Fatalf("expected 5 messages, got %d", len(msgs))
	}
}

func TestLoadRecentSession_EmptyDB(t *testing.T) {
	db := testDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	s, err := LoadRecentSession(db, "telegram", time.Hour)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if s != nil {
		t.Fatalf("expected nil, got %+v", s)
	}
}

func TestLoadRecentSession_RecentSession(t *testing.T) {
	db := testDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	UpsertConversation(db, "tg-abc", "telegram")

	s, err := LoadRecentSession(db, "telegram", time.Hour)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if s == nil {
		t.Fatal("expected session, got nil")
	}
	if s.SessionID != "tg-abc" {
		t.Fatalf("expected tg-abc, got %q", s.SessionID)
	}
}

func TestLoadRecentSession_ExpiredSession(t *testing.T) {
	db := testDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	UpsertConversation(db, "tg-old", "telegram")

	s, err := LoadRecentSession(db, "telegram", 0)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if s != nil {
		t.Fatalf("expected nil for expired session, got %+v", s)
	}
}

func TestLoadRecentSession_MultipleConvos(t *testing.T) {
	db := testDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	UpsertConversation(db, "tg-first", "telegram")
	UpsertConversation(db, "tg-second", "telegram")
	// Make tg-second strictly newer
	db.Exec("UPDATE history_conversations SET updated_at = datetime('now', '+1 second') WHERE session_id = 'tg-second'")

	s, err := LoadRecentSession(db, "telegram", time.Hour)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if s == nil {
		t.Fatal("expected session")
	}
	if s.SessionID != "tg-second" {
		t.Fatalf("expected most recent tg-second, got %q", s.SessionID)
	}
}
