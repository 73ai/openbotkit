package main

import (
	"testing"
)

func TestExtractQueryIDs(t *testing.T) {
	sample := `e.exports={queryId:"abc123def",operationName:"HomeTimeline",operationType:"query"}`
	result := extractQueryIDs(sample)

	if qid, ok := result["HomeTimeline"]; !ok {
		t.Fatal("expected HomeTimeline in results")
	} else if qid != "abc123def" {
		t.Errorf("queryId = %q, want abc123def", qid)
	}
}

func TestExtractQueryIDs_ReverseOrder(t *testing.T) {
	sample := `{operationName:"CreateTweet",queryId:"xyz789"}`
	result := extractQueryIDs(sample)

	if qid, ok := result["CreateTweet"]; !ok {
		t.Fatal("expected CreateTweet in results")
	} else if qid != "xyz789" {
		t.Errorf("queryId = %q, want xyz789", qid)
	}
}

func TestExtractQueryIDs_Multiple(t *testing.T) {
	sample := `
		{queryId:"aaa",operationName:"HomeTimeline"}
		{queryId:"bbb",operationName:"TweetDetail"}
		{queryId:"ccc",operationName:"UnknownOp"}
	`
	result := extractQueryIDs(sample)

	if len(result) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result))
	}
	if result["HomeTimeline"] != "aaa" {
		t.Errorf("HomeTimeline = %q, want aaa", result["HomeTimeline"])
	}
	if result["TweetDetail"] != "bbb" {
		t.Errorf("TweetDetail = %q, want bbb", result["TweetDetail"])
	}
}

func TestExtractChunkHashes(t *testing.T) {
	sample := `"ondemand.s":"a1b2c3d4e5f6","ondemand.s":"fedcba987654"`
	hashes := extractChunkHashes(sample)

	if len(hashes) != 2 {
		t.Fatalf("expected 2 hashes, got %d", len(hashes))
	}
	if hashes[0] != "a1b2c3d4e5f6" {
		t.Errorf("hash[0] = %q, want a1b2c3d4e5f6", hashes[0])
	}
}

func TestExtractChunkHashes_Dedup(t *testing.T) {
	sample := `"ondemand.s":"aabbcc","ondemand.s":"aabbcc"`
	hashes := extractChunkHashes(sample)

	if len(hashes) != 1 {
		t.Fatalf("expected 1 unique hash, got %d", len(hashes))
	}
}

func TestExtractChunkHashes_NoMatches(t *testing.T) {
	hashes := extractChunkHashes("no chunk maps here")
	if len(hashes) != 0 {
		t.Errorf("expected 0 hashes, got %d", len(hashes))
	}
}
