package memory

import "testing"

func TestAdd(t *testing.T) {
	db := testDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	id, err := Add(db, "User prefers dark mode", CategoryPreference, "manual", "")
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero id")
	}

	count, err := Count(db)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1, got %d", count)
	}
}

func TestGet(t *testing.T) {
	db := testDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	id, err := Add(db, "User's name is Priyanshu", CategoryIdentity, "manual", "")
	if err != nil {
		t.Fatalf("add: %v", err)
	}

	m, err := Get(db, id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if m.Content != "User's name is Priyanshu" {
		t.Fatalf("content = %q", m.Content)
	}
	if m.Category != CategoryIdentity {
		t.Fatalf("category = %q", m.Category)
	}
	if m.Source != "manual" {
		t.Fatalf("source = %q", m.Source)
	}
}

func TestGetNotFound(t *testing.T) {
	db := testDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	_, err := Get(db, 99999)
	if err == nil {
		t.Fatal("expected error for non-existent ID")
	}
}

func TestUpdate(t *testing.T) {
	db := testDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	id, err := Add(db, "User prefers light mode", CategoryPreference, "manual", "")
	if err != nil {
		t.Fatalf("add: %v", err)
	}

	if err := Update(db, id, "User prefers dark mode"); err != nil {
		t.Fatalf("update: %v", err)
	}

	m, err := Get(db, id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if m.Content != "User prefers dark mode" {
		t.Fatalf("content = %q, want 'User prefers dark mode'", m.Content)
	}
}

func TestDelete(t *testing.T) {
	db := testDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	id, err := Add(db, "User likes Go", CategoryPreference, "manual", "")
	if err != nil {
		t.Fatalf("add: %v", err)
	}

	if err := Delete(db, id); err != nil {
		t.Fatalf("delete: %v", err)
	}

	count, err := Count(db)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0, got %d", count)
	}
}

func TestList(t *testing.T) {
	db := testDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	Add(db, "User's name is Priyanshu", CategoryIdentity, "manual", "")
	Add(db, "User prefers Go", CategoryPreference, "manual", "")
	Add(db, "User is building OpenBotKit", CategoryProject, "manual", "")

	memories, err := List(db)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(memories) != 3 {
		t.Fatalf("expected 3, got %d", len(memories))
	}
}

func TestListByCategory(t *testing.T) {
	db := testDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	Add(db, "User's name is Priyanshu", CategoryIdentity, "manual", "")
	Add(db, "User prefers Go", CategoryPreference, "manual", "")
	Add(db, "User prefers dark mode", CategoryPreference, "manual", "")

	memories, err := ListByCategory(db, CategoryPreference)
	if err != nil {
		t.Fatalf("list by category: %v", err)
	}
	if len(memories) != 2 {
		t.Fatalf("expected 2 preferences, got %d", len(memories))
	}
}

func TestSearch(t *testing.T) {
	db := testDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	Add(db, "User's name is Priyanshu", CategoryIdentity, "manual", "")
	Add(db, "User prefers Go over Python", CategoryPreference, "manual", "")
	Add(db, "User is building OpenBotKit", CategoryProject, "manual", "")

	memories, err := Search(db, "Go")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(memories) != 1 {
		t.Fatalf("expected 1, got %d", len(memories))
	}
	if memories[0].Content != "User prefers Go over Python" {
		t.Fatalf("content = %q", memories[0].Content)
	}
}

func TestCount(t *testing.T) {
	db := testDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	count, err := Count(db)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0, got %d", count)
	}

	Add(db, "fact one", CategoryIdentity, "manual", "")
	Add(db, "fact two", CategoryPreference, "manual", "")

	count, err = Count(db)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2, got %d", count)
	}
}

func TestAddDuplicateContent(t *testing.T) {
	db := testDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	id1, err := Add(db, "User likes Go", CategoryPreference, "manual", "")
	if err != nil {
		t.Fatalf("first add: %v", err)
	}

	id2, err := Add(db, "User likes Go", CategoryPreference, "manual", "")
	if err != nil {
		t.Fatalf("second add: %v", err)
	}

	if id1 == id2 {
		t.Fatal("expected different IDs for duplicate content")
	}

	count, err := Count(db)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2, got %d", count)
	}
}
