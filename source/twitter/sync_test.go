package twitter

import (
	"context"
	"encoding/json"
	"testing"
)

var emptyTimelineJSON = json.RawMessage(`{"data":{"home":{"home_timeline_urt":{"instructions":[]}}}}`)

type mockFetcher struct {
	homeTimeline       json.RawMessage
	homeLatestTimeline json.RawMessage
	callCount          int
}

func (m *mockFetcher) HomeTimeline(ctx context.Context, count int, cursor string) (json.RawMessage, error) {
	m.callCount++
	if m.callCount > 1 {
		return emptyTimelineJSON, nil
	}
	return m.homeTimeline, nil
}

func (m *mockFetcher) HomeLatestTimeline(ctx context.Context, count int, cursor string) (json.RawMessage, error) {
	m.callCount++
	if m.callCount > 1 {
		return emptyTimelineJSON, nil
	}
	return m.homeLatestTimeline, nil
}

func TestSync_FetchesAndStores(t *testing.T) {
	db := openTestDB(t)

	fetcher := &mockFetcher{homeLatestTimeline: timelineFixture}

	result, err := Sync(context.Background(), db, fetcher, SyncOptions{})
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if result.Fetched != 1 {
		t.Errorf("Fetched = %d, want 1", result.Fetched)
	}

	count, _ := CountTweets(db)
	if count != 1 {
		t.Errorf("tweet count = %d, want 1", count)
	}
}

func TestSync_SkipsDuplicates(t *testing.T) {
	db := openTestDB(t)

	// First sync
	f1 := &mockFetcher{homeLatestTimeline: timelineFixture}
	_, err := Sync(context.Background(), db, f1, SyncOptions{})
	if err != nil {
		t.Fatalf("first sync: %v", err)
	}

	// Second sync with fresh fetcher — should skip existing
	f2 := &mockFetcher{homeLatestTimeline: timelineFixture}
	result, err := Sync(context.Background(), db, f2, SyncOptions{})
	if err != nil {
		t.Fatalf("second sync: %v", err)
	}
	if result.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", result.Skipped)
	}
	if result.Fetched != 0 {
		t.Errorf("Fetched = %d, want 0", result.Fetched)
	}
}

func TestSync_FullSync(t *testing.T) {
	db := openTestDB(t)

	// Initial sync
	f1 := &mockFetcher{homeLatestTimeline: timelineFixture}
	_, err := Sync(context.Background(), db, f1, SyncOptions{})
	if err != nil {
		t.Fatalf("first sync: %v", err)
	}

	// Full sync with fresh fetcher — should re-save (upsert)
	f2 := &mockFetcher{homeLatestTimeline: timelineFixture}
	result, err := Sync(context.Background(), db, f2, SyncOptions{Full: true})
	if err != nil {
		t.Fatalf("full sync: %v", err)
	}
	if result.Fetched != 1 {
		t.Errorf("Fetched = %d, want 1 (full sync upserts)", result.Fetched)
	}
}

func TestSync_ForYouTimeline(t *testing.T) {
	db := openTestDB(t)

	fetcher := &mockFetcher{homeTimeline: timelineFixture}

	result, err := Sync(context.Background(), db, fetcher, SyncOptions{TimelineType: "foryou"})
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if result.Fetched != 1 {
		t.Errorf("Fetched = %d, want 1", result.Fetched)
	}
}

func TestSync_SavesCursor(t *testing.T) {
	db := openTestDB(t)

	fetcher := &mockFetcher{homeLatestTimeline: timelineFixture}

	_, err := Sync(context.Background(), db, fetcher, SyncOptions{})
	if err != nil {
		t.Fatalf("sync: %v", err)
	}

	state, err := GetSyncState(db, "following")
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	if state == nil {
		t.Fatal("expected sync state, got nil")
	}
	if state.Cursor != "next-cursor-abc" {
		t.Errorf("Cursor = %q, want next-cursor-abc", state.Cursor)
	}
}

func TestSync_EmptyTimeline(t *testing.T) {
	db := openTestDB(t)

	fetcher := &mockFetcher{
		homeLatestTimeline: json.RawMessage(`{"data":{"home":{"home_timeline_urt":{"instructions":[]}}}}`),
	}

	result, err := Sync(context.Background(), db, fetcher, SyncOptions{})
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if result.Fetched != 0 {
		t.Errorf("Fetched = %d, want 0", result.Fetched)
	}
}
