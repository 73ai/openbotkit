package twitter

import (
	"context"
	"testing"
)

func TestNew(t *testing.T) {
	x := New(Config{})
	if x == nil {
		t.Fatal("expected non-nil X")
	}
}

func TestName(t *testing.T) {
	x := New(Config{})
	if x.Name() != "x" {
		t.Errorf("Name() = %q, want x", x.Name())
	}
}

func TestStatus_EmptyDB(t *testing.T) {
	db := openTestDB(t)
	x := New(Config{})

	status, err := x.Status(context.Background(), db)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.ItemCount != 0 {
		t.Errorf("ItemCount = %d, want 0", status.ItemCount)
	}
	if status.LastSyncedAt != nil {
		t.Errorf("LastSyncedAt should be nil for empty DB")
	}
}

func TestStatus_WithData(t *testing.T) {
	db := openTestDB(t)
	x := New(Config{})

	if err := SaveTweet(db, sampleTweet("st1")); err != nil {
		t.Fatalf("save: %v", err)
	}

	status, err := x.Status(context.Background(), db)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.ItemCount != 1 {
		t.Errorf("ItemCount = %d, want 1", status.ItemCount)
	}
}

func TestXSync_DelegatesToPackageSync(t *testing.T) {
	db := openTestDB(t)
	x := New(Config{})

	fetcher := &mockFetcher{homeLatestTimeline: timelineFixture}
	result, err := x.Sync(context.Background(), db, fetcher, SyncOptions{})
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if result.Fetched != 1 {
		t.Errorf("Fetched = %d, want 1", result.Fetched)
	}
}
