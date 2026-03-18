package contacts

import (
	"testing"
	"time"

	"github.com/73ai/openbotkit/store"
)

func seedContact(t *testing.T, db *store.DB, name string, aliases []string, interactions map[string]int) int64 {
	t.Helper()
	id, err := CreateContact(db, name)
	if err != nil {
		t.Fatalf("create %s: %v", name, err)
	}
	for _, a := range aliases {
		if err := AddAlias(db, id, a, "test"); err != nil {
			t.Fatalf("alias %s: %v", a, err)
		}
	}
	now := time.Now()
	for ch, count := range interactions {
		if err := UpsertInteraction(db, id, ch, count, &now); err != nil {
			t.Fatalf("interaction: %v", err)
		}
	}
	return id
}

func TestSearchExactMatch(t *testing.T) {
	db := testDB(t)
	seedContact(t, db, "David Chen", []string{"David Chen", "David"}, map[string]int{"whatsapp": 5})

	results, err := SearchContacts(db, "David Chen", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Contact.DisplayName != "David Chen" {
		t.Errorf("name = %q", results[0].Contact.DisplayName)
	}
}

func TestSearchPartialName(t *testing.T) {
	db := testDB(t)
	seedContact(t, db, "David Chen", []string{"David Chen", "David"}, nil)

	results, err := SearchContacts(db, "Dav", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
}

func TestSearchCaseInsensitive(t *testing.T) {
	db := testDB(t)
	seedContact(t, db, "Alice Smith", []string{"Alice Smith"}, nil)

	results, err := SearchContacts(db, "alice", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
}

func TestSearchByAlias(t *testing.T) {
	db := testDB(t)
	seedContact(t, db, "Robert Johnson", []string{"Robert Johnson", "Bob"}, nil)

	results, err := SearchContacts(db, "Bob", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d, want 1", len(results))
	}
	if results[0].MatchedAlias != "Bob" {
		t.Errorf("matched alias = %q, want Bob", results[0].MatchedAlias)
	}
}

func TestSearchRankingByFrequency(t *testing.T) {
	db := testDB(t)
	seedContact(t, db, "David Low", []string{"David Low", "David"}, map[string]int{"whatsapp": 5})
	seedContact(t, db, "David High", []string{"David High", "David"}, map[string]int{"whatsapp": 100})

	results, err := SearchContacts(db, "David", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	if results[0].Contact.DisplayName != "David High" {
		t.Errorf("first result = %q, want David High (higher frequency)", results[0].Contact.DisplayName)
	}
}

func TestSearchPrefixRanksHigher(t *testing.T) {
	db := testDB(t)
	seedContact(t, db, "Sandra Lee", []string{"Sandra Lee"}, map[string]int{"gmail": 100})
	seedContact(t, db, "San Patel", []string{"San Patel"}, map[string]int{"gmail": 1})

	results, err := SearchContacts(db, "San", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) < 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	// Both match as prefix, so higher frequency wins.
	if results[0].Contact.DisplayName != "Sandra Lee" {
		t.Errorf("first = %q, want Sandra Lee", results[0].Contact.DisplayName)
	}
}

func TestSearchNoResults(t *testing.T) {
	db := testDB(t)
	seedContact(t, db, "Alice", []string{"Alice"}, nil)

	results, err := SearchContacts(db, "zzzzz", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Errorf("got %d results, want 0", len(results))
	}
}

func TestSearchEmptyQuery(t *testing.T) {
	db := testDB(t)
	results, err := SearchContacts(db, "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if results != nil {
		t.Errorf("expected nil for empty query")
	}
}

func TestSearchUnicode(t *testing.T) {
	db := testDB(t)
	seedContact(t, db, "José García", []string{"José García"}, nil)

	results, err := SearchContacts(db, "José", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d, want 1", len(results))
	}
}

func TestSearchSpecialChars(t *testing.T) {
	db := testDB(t)
	seedContact(t, db, "O'Brien", []string{"O'Brien"}, nil)

	results, err := SearchContacts(db, "O'Br", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d, want 1", len(results))
	}
}

func TestSearchSubstring(t *testing.T) {
	db := testDB(t)
	seedContact(t, db, "Smith-Jones", []string{"Smith-Jones"}, nil)

	results, err := SearchContacts(db, "Jones", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d, want 1", len(results))
	}
}
