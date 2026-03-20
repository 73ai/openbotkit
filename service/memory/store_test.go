package memory

import "testing"

func TestAdd(t *testing.T) {
	s := testStore(t)

	id, err := s.Add("User prefers dark mode", CategoryPreference, "manual", "")
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero id")
	}

	count, err := s.Count()
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1, got %d", count)
	}
}

func TestGet(t *testing.T) {
	s := testStore(t)

	id, err := s.Add("User's name is Priyanshu", CategoryIdentity, "manual", "")
	if err != nil {
		t.Fatalf("add: %v", err)
	}

	m, err := s.Get(id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if m.Content != "User's name is Priyanshu" {
		t.Fatalf("content = %q", m.Content)
	}
	if m.Category != CategoryIdentity {
		t.Fatalf("category = %q", m.Category)
	}
}

func TestGetNotFound(t *testing.T) {
	s := testStore(t)

	_, err := s.Get(99999)
	if err == nil {
		t.Fatal("expected error for non-existent ID")
	}
}

func TestUpdate(t *testing.T) {
	s := testStore(t)

	id, err := s.Add("User prefers light mode", CategoryPreference, "manual", "")
	if err != nil {
		t.Fatalf("add: %v", err)
	}

	if err := s.Update(id, "User prefers dark mode"); err != nil {
		t.Fatalf("update: %v", err)
	}

	m, err := s.Get(id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if m.Content != "User prefers dark mode" {
		t.Fatalf("content = %q, want 'User prefers dark mode'", m.Content)
	}
}

func TestDelete(t *testing.T) {
	s := testStore(t)

	id, err := s.Add("User likes Go", CategoryPreference, "manual", "")
	if err != nil {
		t.Fatalf("add: %v", err)
	}

	if err := s.Delete(id); err != nil {
		t.Fatalf("delete: %v", err)
	}

	count, err := s.Count()
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0, got %d", count)
	}
}

func TestList(t *testing.T) {
	s := testStore(t)

	s.Add("User's name is Priyanshu", CategoryIdentity, "manual", "")
	s.Add("User prefers Go", CategoryPreference, "manual", "")
	s.Add("User is building OpenBotKit", CategoryProject, "manual", "")

	memories, err := s.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(memories) != 3 {
		t.Fatalf("expected 3, got %d", len(memories))
	}
}

func TestListByCategory(t *testing.T) {
	s := testStore(t)

	s.Add("User's name is Priyanshu", CategoryIdentity, "manual", "")
	s.Add("User prefers Go", CategoryPreference, "manual", "")
	s.Add("User prefers dark mode", CategoryPreference, "manual", "")

	memories, err := s.ListByCategory(CategoryPreference)
	if err != nil {
		t.Fatalf("list by category: %v", err)
	}
	if len(memories) != 2 {
		t.Fatalf("expected 2 preferences, got %d", len(memories))
	}
}

func TestSearch(t *testing.T) {
	s := testStore(t)

	s.Add("User's name is Priyanshu", CategoryIdentity, "manual", "")
	s.Add("User prefers Go over Python", CategoryPreference, "manual", "")
	s.Add("User is building OpenBotKit", CategoryProject, "manual", "")

	memories, err := s.Search("Go")
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
	s := testStore(t)

	count, err := s.Count()
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0, got %d", count)
	}

	s.Add("fact one", CategoryIdentity, "manual", "")
	s.Add("fact two", CategoryPreference, "manual", "")

	count, err = s.Count()
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2, got %d", count)
	}
}

func TestAddDuplicateContent(t *testing.T) {
	s := testStore(t)

	id1, err := s.Add("User likes Go", CategoryPreference, "manual", "")
	if err != nil {
		t.Fatalf("first add: %v", err)
	}

	id2, err := s.Add("User likes Go", CategoryPreference, "manual", "")
	if err != nil {
		t.Fatalf("second add: %v", err)
	}

	if id1 == id2 {
		t.Fatal("expected different IDs for duplicate content")
	}

	count, err := s.Count()
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2, got %d", count)
	}
}

func TestIDGloballyUnique(t *testing.T) {
	s := testStore(t)

	id1, _ := s.Add("fact one", CategoryIdentity, "manual", "")
	id2, _ := s.Add("fact two", CategoryPreference, "manual", "")
	id3, _ := s.Add("fact three", CategoryProject, "manual", "")

	if id1 == id2 || id2 == id3 || id1 == id3 {
		t.Fatalf("IDs should be unique across categories: %d, %d, %d", id1, id2, id3)
	}
}
