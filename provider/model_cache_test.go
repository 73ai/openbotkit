package provider

import (
	"testing"
	"time"
)

func TestModelCache_SaveLoad(t *testing.T) {
	cache := NewModelCache(t.TempDir())
	list := &CachedModelList{
		Provider:  "openai",
		FetchedAt: time.Now(),
		Models: []AvailableModel{
			{ID: "gpt-4o", DisplayName: "GPT-4o", Provider: "openai"},
		},
	}

	if err := cache.Save("openai", list); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := cache.Load("openai")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(loaded.Models) != 1 || loaded.Models[0].ID != "gpt-4o" {
		t.Errorf("unexpected loaded models: %+v", loaded.Models)
	}
}

func TestModelCache_LoadMissing(t *testing.T) {
	cache := NewModelCache(t.TempDir())
	_, err := cache.Load("nonexistent")
	if err == nil {
		t.Fatal("expected error loading nonexistent cache")
	}
}

func TestModelCache_IsStale(t *testing.T) {
	dir := t.TempDir()
	cache := NewModelCache(dir)

	// Non-existent file is stale.
	if !cache.IsStale("openai", 24*time.Hour) {
		t.Error("non-existent cache should be stale")
	}

	// Save and check — should not be stale.
	list := &CachedModelList{Provider: "openai", FetchedAt: time.Now()}
	if err := cache.Save("openai", list); err != nil {
		t.Fatalf("save: %v", err)
	}
	if cache.IsStale("openai", 24*time.Hour) {
		t.Error("freshly saved cache should not be stale")
	}

	// Zero TTL means always stale.
	if !cache.IsStale("openai", 0) {
		t.Error("zero TTL should be stale")
	}
}

func TestModelCache_VerifyModel(t *testing.T) {
	cache := NewModelCache(t.TempDir())
	list := &CachedModelList{
		Provider:  "openai",
		FetchedAt: time.Now(),
		Models:    []AvailableModel{{ID: "gpt-4o", Provider: "openai"}},
	}
	if err := cache.Save("openai", list); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Not verified yet.
	if cache.IsModelVerified("openai", "gpt-4o", "key-1") {
		t.Error("model should not be verified yet")
	}

	// Mark verified.
	if err := cache.MarkModelVerified("openai", "gpt-4o", "key-1"); err != nil {
		t.Fatalf("mark verified: %v", err)
	}

	// Now verified.
	if !cache.IsModelVerified("openai", "gpt-4o", "key-1") {
		t.Error("model should be verified after marking")
	}

	// Different key — not verified.
	if cache.IsModelVerified("openai", "gpt-4o", "key-2") {
		t.Error("model should not be verified with different key")
	}
}

func TestModelCache_MarkVerified_NoExistingCache(t *testing.T) {
	cache := NewModelCache(t.TempDir())

	// Should create the file.
	if err := cache.MarkModelVerified("anthropic", "claude-3", "key-1"); err != nil {
		t.Fatalf("mark verified: %v", err)
	}
	if !cache.IsModelVerified("anthropic", "claude-3", "key-1") {
		t.Error("should be verified even without prior cache")
	}
}

func TestHashKey_Deterministic(t *testing.T) {
	h1 := hashKey("test-key")
	h2 := hashKey("test-key")
	if h1 != h2 {
		t.Errorf("hash should be deterministic: %s != %s", h1, h2)
	}
	h3 := hashKey("different-key")
	if h1 == h3 {
		t.Error("different keys should produce different hashes")
	}
}
