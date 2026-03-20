package history

import (
	"fmt"
	"testing"
	"time"
)

func TestUpsertConversation(t *testing.T) {
	s := testStore(t)

	if err := s.UpsertConversation("session-001", "/tmp/project"); err != nil {
		t.Fatalf("first upsert: %v", err)
	}

	// Upsert same session should succeed.
	if err := s.UpsertConversation("session-001", "/tmp/project"); err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	// Different session should also succeed.
	if err := s.UpsertConversation("session-002", "/tmp/other"); err != nil {
		t.Fatalf("third upsert: %v", err)
	}

	count, _ := s.CountConversations()
	if count != 2 {
		t.Fatalf("expected 2 conversations, got %d", count)
	}
}

func TestSaveMessage(t *testing.T) {
	s := testStore(t)
	s.UpsertConversation("session-001", "/tmp")

	if err := s.SaveMessage("session-001", "user", "hello"); err != nil {
		t.Fatalf("save user message: %v", err)
	}
	if err := s.SaveMessage("session-001", "assistant", "hi there"); err != nil {
		t.Fatalf("save assistant message: %v", err)
	}

	count, err := s.MessageCountForSession("session-001")
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 messages, got %d", count)
	}
}

func TestCountConversations(t *testing.T) {
	s := testStore(t)

	count, err := s.CountConversations()
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0, got %d", count)
	}

	s.UpsertConversation("s1", "/tmp")
	s.UpsertConversation("s2", "/tmp")

	count, err = s.CountConversations()
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2, got %d", count)
	}
}

func TestLastCaptureTime(t *testing.T) {
	s := testStore(t)

	ts, err := s.LastCaptureTime()
	if err != nil {
		t.Fatalf("last capture: %v", err)
	}
	if ts != nil {
		t.Fatalf("expected nil, got %v", ts)
	}

	s.UpsertConversation("s1", "/tmp")

	ts, err = s.LastCaptureTime()
	if err != nil {
		t.Fatalf("last capture: %v", err)
	}
	if ts == nil {
		t.Fatal("expected non-nil timestamp after upsert")
	}
}

func TestMessageCountForSession(t *testing.T) {
	s := testStore(t)

	count, err := s.MessageCountForSession("nonexistent")
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0, got %d", count)
	}
}

func TestLoadSessionMessages_RoundTrip(t *testing.T) {
	s := testStore(t)
	s.UpsertConversation("tg-msg", "telegram")
	s.SaveMessage("tg-msg", "user", "hello")
	s.SaveMessage("tg-msg", "assistant", "hi there")
	s.SaveMessage("tg-msg", "user", "bye")

	msgs, err := s.LoadSessionMessages("tg-msg", 100)
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
	s := testStore(t)
	msgs, err := s.LoadSessionMessages("nonexistent", 100)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(msgs) != 0 {
		t.Fatalf("expected 0 messages, got %d", len(msgs))
	}
}

func TestLoadSessionMessages_Limit(t *testing.T) {
	s := testStore(t)
	s.UpsertConversation("tg-limit", "telegram")
	for i := range 10 {
		s.SaveMessage("tg-limit", "user", fmt.Sprintf("msg %d", i))
	}

	msgs, err := s.LoadSessionMessages("tg-limit", 5)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(msgs) != 5 {
		t.Fatalf("expected 5 messages, got %d", len(msgs))
	}
	// Should return the LAST 5 messages (5-9), not the first 5.
	if msgs[0].Content != "msg 5" {
		t.Errorf("expected last 5 msgs starting with 'msg 5', got %q", msgs[0].Content)
	}
	if msgs[4].Content != "msg 9" {
		t.Errorf("expected last msg 'msg 9', got %q", msgs[4].Content)
	}
}

func TestLoadRecentSession_EmptyDB(t *testing.T) {
	s := testStore(t)
	rs, err := s.LoadRecentSession("telegram", time.Hour)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if rs != nil {
		t.Fatalf("expected nil, got %+v", rs)
	}
}

func TestLoadRecentSession_RecentSession(t *testing.T) {
	s := testStore(t)
	s.UpsertConversation("tg-abc", "telegram")

	rs, err := s.LoadRecentSession("telegram", time.Hour)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if rs == nil {
		t.Fatal("expected session, got nil")
	}
	if rs.SessionID != "tg-abc" {
		t.Fatalf("expected tg-abc, got %q", rs.SessionID)
	}
}

func TestLoadRecentSession_ExpiredSession(t *testing.T) {
	s := testStore(t)
	s.UpsertConversation("tg-old", "telegram")

	rs, err := s.LoadRecentSession("telegram", 0)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if rs != nil {
		t.Fatalf("expected nil for expired session, got %+v", rs)
	}
}

func TestLoadRecentSession_MultipleConvos(t *testing.T) {
	s := testStore(t)
	s.UpsertConversation("tg-first", "telegram")
	// Sleep to ensure tg-second gets a strictly newer timestamp.
	time.Sleep(time.Millisecond)
	s.UpsertConversation("tg-second", "telegram")

	rs, err := s.LoadRecentSession("telegram", time.Hour)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if rs == nil {
		t.Fatal("expected session")
	}
	if rs.SessionID != "tg-second" {
		t.Fatalf("expected most recent tg-second, got %q", rs.SessionID)
	}
}

func TestEndSession_ExcludedFromRestore(t *testing.T) {
	s := testStore(t)
	s.UpsertConversation("tg-ended", "telegram")
	if err := s.EndSession("tg-ended"); err != nil {
		t.Fatalf("end session: %v", err)
	}

	rs, err := s.LoadRecentSession("telegram", time.Hour)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if rs != nil {
		t.Fatalf("expected nil after ending session, got %+v", rs)
	}
}

func TestPathTraversal_Rejected(t *testing.T) {
	s := testStore(t)

	for _, bad := range []string{"../../etc/passwd", "../escape", "a/b/c", "hello world", ""} {
		if err := s.UpsertConversation(bad, "cli"); err == nil {
			t.Errorf("expected error for sessionID %q", bad)
		}
		if err := s.SaveMessage(bad, "user", "hi"); err == nil {
			t.Errorf("expected error for sessionID %q", bad)
		}
	}
}

func TestLoadRecentUserMessages(t *testing.T) {
	s := testStore(t)
	s.UpsertConversation("s1", "cli")
	s.SaveMessage("s1", "user", "hello")
	s.SaveMessage("s1", "assistant", "hi")
	s.SaveMessage("s1", "user", "bye")

	msgs, err := s.LoadRecentUserMessages(1)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 user messages, got %d", len(msgs))
	}
	if msgs[0] != "hello" {
		t.Errorf("msg[0] = %q, want hello", msgs[0])
	}
	if msgs[1] != "bye" {
		t.Errorf("msg[1] = %q, want bye", msgs[1])
	}
}
