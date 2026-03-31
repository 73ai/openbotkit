package twitter

import (
	"fmt"
	"testing"
	"time"
)

func sampleTweet(id string) *Tweet {
	return &Tweet{
		TweetID:        id,
		UserID:         "user123",
		UserName:       "testuser",
		UserFullName:   "Test User",
		Text:           "Hello from X! #testing",
		CreatedAt:      time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC),
		ConversationID: id,
		RetweetCount:   5,
		LikeCount:      10,
		ReplyCount:     2,
	}
}

func TestSaveTweet_AndGetTweet(t *testing.T) {
	db := openTestDB(t)

	tweet := sampleTweet("1234567890")
	if err := SaveTweet(db, tweet); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := GetTweet(db, "1234567890")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got == nil {
		t.Fatal("expected tweet, got nil")
	}
	if got.TweetID != "1234567890" {
		t.Errorf("TweetID = %q, want 1234567890", got.TweetID)
	}
	if got.Text != "Hello from X! #testing" {
		t.Errorf("Text = %q, want 'Hello from X! #testing'", got.Text)
	}
	if got.LikeCount != 10 {
		t.Errorf("LikeCount = %d, want 10", got.LikeCount)
	}
}

func TestSaveTweet_Upsert(t *testing.T) {
	db := openTestDB(t)

	tweet := sampleTweet("111")
	if err := SaveTweet(db, tweet); err != nil {
		t.Fatalf("first save: %v", err)
	}

	tweet.LikeCount = 99
	tweet.RetweetCount = 50
	if err := SaveTweet(db, tweet); err != nil {
		t.Fatalf("second save: %v", err)
	}

	got, err := GetTweet(db, "111")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.LikeCount != 99 {
		t.Errorf("LikeCount = %d, want 99 after upsert", got.LikeCount)
	}
	if got.RetweetCount != 50 {
		t.Errorf("RetweetCount = %d, want 50 after upsert", got.RetweetCount)
	}
}

func TestTweetExists(t *testing.T) {
	db := openTestDB(t)

	exists, err := TweetExists(db, "nonexistent")
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if exists {
		t.Error("expected false for nonexistent tweet")
	}

	if err := SaveTweet(db, sampleTweet("222")); err != nil {
		t.Fatalf("save: %v", err)
	}

	exists, err = TweetExists(db, "222")
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if !exists {
		t.Error("expected true for existing tweet")
	}
}

func TestGetTweet_NotFound(t *testing.T) {
	db := openTestDB(t)

	got, err := GetTweet(db, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
}

func TestListTweets(t *testing.T) {
	db := openTestDB(t)

	for i := 0; i < 5; i++ {
		tw := sampleTweet(fmt.Sprintf("tw%d", i))
		tw.CreatedAt = time.Date(2026, 3, 29, i, 0, 0, 0, time.UTC)
		if err := SaveTweet(db, tw); err != nil {
			t.Fatalf("save tweet %d: %v", i, err)
		}
	}

	tweets, err := ListTweets(db, ListOptions{Limit: 3})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(tweets) != 3 {
		t.Fatalf("expected 3 tweets, got %d", len(tweets))
	}
	if tweets[0].CreatedAt.Before(tweets[1].CreatedAt) {
		t.Error("expected newest first ordering")
	}
}

func TestListTweets_WithOffset(t *testing.T) {
	db := openTestDB(t)

	for i := 0; i < 5; i++ {
		tw := sampleTweet(fmt.Sprintf("off%d", i))
		tw.CreatedAt = time.Date(2026, 3, 29, i, 0, 0, 0, time.UTC)
		if err := SaveTweet(db, tw); err != nil {
			t.Fatalf("save tweet %d: %v", i, err)
		}
	}

	tweets, err := ListTweets(db, ListOptions{Limit: 2, Offset: 2})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(tweets) != 2 {
		t.Fatalf("expected 2 tweets, got %d", len(tweets))
	}
}

func TestSearchTweets(t *testing.T) {
	db := openTestDB(t)

	tw1 := sampleTweet("s1")
	tw1.Text = "Golang is awesome"
	tw2 := sampleTweet("s2")
	tw2.Text = "Python is great"
	tw3 := sampleTweet("s3")
	tw3.Text = "Learning golang today"

	for _, tw := range []*Tweet{tw1, tw2, tw3} {
		if err := SaveTweet(db, tw); err != nil {
			t.Fatalf("save: %v", err)
		}
	}

	results, err := SearchTweets(db, "golang", 50)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results for 'golang', got %d", len(results))
	}
}

func TestCountTweets(t *testing.T) {
	db := openTestDB(t)

	count, err := CountTweets(db)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}

	if err := SaveTweet(db, sampleTweet("c1")); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := SaveTweet(db, sampleTweet("c2")); err != nil {
		t.Fatalf("save: %v", err)
	}

	count, err = CountTweets(db)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}
}

func TestSyncState(t *testing.T) {
	db := openTestDB(t)

	state, err := GetSyncState(db, "foryou")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if state != nil {
		t.Fatalf("expected nil, got %+v", state)
	}

	if err := SaveSyncState(db, "foryou", "cursor-abc"); err != nil {
		t.Fatalf("save: %v", err)
	}

	state, err = GetSyncState(db, "foryou")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if state == nil {
		t.Fatal("expected state, got nil")
	}
	if state.Cursor != "cursor-abc" {
		t.Errorf("Cursor = %q, want cursor-abc", state.Cursor)
	}

	if err := SaveSyncState(db, "foryou", "cursor-xyz"); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	state, err = GetSyncState(db, "foryou")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if state.Cursor != "cursor-xyz" {
		t.Errorf("Cursor = %q, want cursor-xyz after upsert", state.Cursor)
	}
}

func TestSyncState_MultipleTimelines(t *testing.T) {
	db := openTestDB(t)

	if err := SaveSyncState(db, "foryou", "c1"); err != nil {
		t.Fatalf("save foryou: %v", err)
	}
	if err := SaveSyncState(db, "following", "c2"); err != nil {
		t.Fatalf("save following: %v", err)
	}

	s1, err := GetSyncState(db, "foryou")
	if err != nil {
		t.Fatalf("get foryou: %v", err)
	}
	s2, err := GetSyncState(db, "following")
	if err != nil {
		t.Fatalf("get following: %v", err)
	}

	if s1.Cursor != "c1" {
		t.Errorf("foryou cursor = %q, want c1", s1.Cursor)
	}
	if s2.Cursor != "c2" {
		t.Errorf("following cursor = %q, want c2", s2.Cursor)
	}
}

func TestLastSyncTime_Empty(t *testing.T) {
	db := openTestDB(t)

	ts, err := LastSyncTime(db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ts != nil {
		t.Errorf("expected nil for empty DB, got %v", ts)
	}
}

func TestLastSyncTime_WithData(t *testing.T) {
	db := openTestDB(t)

	if err := SaveTweet(db, sampleTweet("ls1")); err != nil {
		t.Fatalf("save: %v", err)
	}

	ts, err := LastSyncTime(db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ts == nil {
		t.Fatal("expected non-nil timestamp after saving tweet")
	}
}

func TestListTweets_DefaultLimit(t *testing.T) {
	db := openTestDB(t)

	if err := SaveTweet(db, sampleTweet("dl1")); err != nil {
		t.Fatalf("save: %v", err)
	}

	tweets, err := ListTweets(db, ListOptions{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(tweets) != 1 {
		t.Errorf("expected 1 tweet, got %d", len(tweets))
	}
}

func TestSearchTweets_NoResults(t *testing.T) {
	db := openTestDB(t)

	if err := SaveTweet(db, sampleTweet("sr1")); err != nil {
		t.Fatalf("save: %v", err)
	}

	results, err := SearchTweets(db, "nonexistentterm", 50)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestSearchTweets_ByUsername(t *testing.T) {
	db := openTestDB(t)

	tw := sampleTweet("su1")
	tw.UserName = "specialuser"
	if err := SaveTweet(db, tw); err != nil {
		t.Fatalf("save: %v", err)
	}

	results, err := SearchTweets(db, "specialuser", 50)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result for username search, got %d", len(results))
	}
}
