package contacts

import (
	"testing"
	"time"
)

func TestCreateAndGetContact(t *testing.T) {
	db := testDB(t)
	id, err := CreateContact(db, "Alice Smith")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	c, err := GetContact(db, id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if c == nil {
		t.Fatal("contact not found")
	}
	if c.DisplayName != "Alice Smith" {
		t.Errorf("name = %q, want %q", c.DisplayName, "Alice Smith")
	}
}

func TestGetContactNotFound(t *testing.T) {
	db := testDB(t)
	c, err := GetContact(db, 99999)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if c != nil {
		t.Error("expected nil for non-existent contact")
	}
}

func TestDeleteContact(t *testing.T) {
	db := testDB(t)
	id, err := CreateContact(db, "Delete Me")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := AddAlias(db, id, "DM", "test"); err != nil {
		t.Fatalf("add alias: %v", err)
	}
	if err := DeleteContact(db, id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	c, err := GetContact(db, id)
	if err != nil {
		t.Fatalf("get after delete: %v", err)
	}
	if c != nil {
		t.Error("contact should be nil after delete")
	}
}

func TestListContacts(t *testing.T) {
	db := testDB(t)
	for _, name := range []string{"Bob", "Alice", "Charlie"} {
		if _, err := CreateContact(db, name); err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
	}
	contacts, err := ListContacts(db, 10, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(contacts) != 3 {
		t.Fatalf("got %d contacts, want 3", len(contacts))
	}
	// Should be sorted by display_name.
	if contacts[0].DisplayName != "Alice" {
		t.Errorf("first = %q, want Alice", contacts[0].DisplayName)
	}
}

func TestListContactsWithLimit(t *testing.T) {
	db := testDB(t)
	for _, name := range []string{"A", "B", "C"} {
		if _, err := CreateContact(db, name); err != nil {
			t.Fatalf("create: %v", err)
		}
	}
	contacts, err := ListContacts(db, 2, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(contacts) != 2 {
		t.Fatalf("got %d, want 2", len(contacts))
	}
}

func TestCountContacts(t *testing.T) {
	db := testDB(t)
	if _, err := CreateContact(db, "One"); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateContact(db, "Two"); err != nil {
		t.Fatal(err)
	}
	count, err := CountContacts(db)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestUpsertIdentity(t *testing.T) {
	db := testDB(t)
	id, _ := CreateContact(db, "Test")
	err := UpsertIdentity(db, &Identity{
		ContactID:     id,
		Source:        "whatsapp",
		IdentityType:  "phone",
		IdentityValue: "+1234",
		DisplayName:   "Test",
		RawValue:      "1234",
	})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	c, _ := GetContact(db, id)
	if len(c.Identities) != 1 {
		t.Fatalf("got %d identities, want 1", len(c.Identities))
	}
	if c.Identities[0].IdentityValue != "+1234" {
		t.Errorf("value = %q, want +1234", c.Identities[0].IdentityValue)
	}
}

func TestUpsertIdentityDuplicate(t *testing.T) {
	db := testDB(t)
	id, _ := CreateContact(db, "Test")
	ident := &Identity{
		ContactID:     id,
		Source:        "whatsapp",
		IdentityType:  "phone",
		IdentityValue: "+1234",
		DisplayName:   "Old",
	}
	if err := UpsertIdentity(db, ident); err != nil {
		t.Fatal(err)
	}
	ident.DisplayName = "New"
	if err := UpsertIdentity(db, ident); err != nil {
		t.Fatal(err)
	}
	c, _ := GetContact(db, id)
	if len(c.Identities) != 1 {
		t.Fatalf("got %d identities, want 1", len(c.Identities))
	}
	if c.Identities[0].DisplayName != "New" {
		t.Errorf("display_name = %q, want New", c.Identities[0].DisplayName)
	}
}

func TestFindContactByIdentity(t *testing.T) {
	db := testDB(t)
	id, _ := CreateContact(db, "Found")
	_ = UpsertIdentity(db, &Identity{
		ContactID: id, Source: "gmail", IdentityType: "email", IdentityValue: "found@test.com",
	})
	c, err := FindContactByIdentity(db, "email", "found@test.com")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if c == nil || c.ID != id {
		t.Errorf("expected contact %d, got %v", id, c)
	}

	c, err = FindContactByIdentity(db, "email", "nope@test.com")
	if err != nil {
		t.Fatalf("find missing: %v", err)
	}
	if c != nil {
		t.Error("expected nil for non-existent identity")
	}
}

func TestAddAlias(t *testing.T) {
	db := testDB(t)
	id, _ := CreateContact(db, "Ali")
	if err := AddAlias(db, id, "Ali B", "test"); err != nil {
		t.Fatal(err)
	}
	if err := AddAlias(db, id, "Ali B", "test"); err != nil {
		t.Fatal("duplicate alias should not error")
	}
	if err := AddAlias(db, id, "", "test"); err != nil {
		t.Fatal("empty alias should be no-op")
	}
	c, _ := GetContact(db, id)
	if len(c.Aliases) != 1 {
		t.Errorf("got %d aliases, want 1", len(c.Aliases))
	}
}

func TestUpsertInteraction(t *testing.T) {
	db := testDB(t)
	id, _ := CreateContact(db, "Inter")
	now := time.Now()
	if err := UpsertInteraction(db, id, "whatsapp", 10, &now); err != nil {
		t.Fatal(err)
	}
	if err := UpsertInteraction(db, id, "whatsapp", 20, &now); err != nil {
		t.Fatal(err)
	}
	c, _ := GetContact(db, id)
	if len(c.Interactions) != 1 {
		t.Fatalf("got %d interactions, want 1", len(c.Interactions))
	}
	if c.Interactions[0].MessageCount != 20 {
		t.Errorf("count = %d, want 20", c.Interactions[0].MessageCount)
	}
}

func TestMergeContacts(t *testing.T) {
	db := testDB(t)
	keepID, _ := CreateContact(db, "Keep")
	mergeID, _ := CreateContact(db, "Merge")

	_ = UpsertIdentity(db, &Identity{ContactID: keepID, Source: "wa", IdentityType: "phone", IdentityValue: "+1"})
	_ = UpsertIdentity(db, &Identity{ContactID: mergeID, Source: "gmail", IdentityType: "email", IdentityValue: "m@test.com"})
	_ = AddAlias(db, mergeID, "Merge Name", "test")

	if err := MergeContacts(db, keepID, mergeID); err != nil {
		t.Fatalf("merge: %v", err)
	}

	kept, _ := GetContact(db, keepID)
	if len(kept.Identities) != 2 {
		t.Errorf("kept identities = %d, want 2", len(kept.Identities))
	}
	if len(kept.Aliases) != 1 {
		t.Errorf("kept aliases = %d, want 1", len(kept.Aliases))
	}

	merged, _ := GetContact(db, mergeID)
	if merged != nil {
		t.Error("merged contact should be deleted")
	}
}

func TestMergeContactsWithConflicts(t *testing.T) {
	db := testDB(t)
	keepID, _ := CreateContact(db, "Keep")
	mergeID, _ := CreateContact(db, "Merge")

	// Both contacts share the same phone identity (UNIQUE conflict on merge).
	_ = UpsertIdentity(db, &Identity{ContactID: keepID, Source: "wa", IdentityType: "phone", IdentityValue: "+1"})
	_ = UpsertIdentity(db, &Identity{ContactID: mergeID, Source: "wa", IdentityType: "phone", IdentityValue: "+1"})
	// Both have the same alias (UNIQUE conflict on merge).
	_ = AddAlias(db, keepID, "Same Name", "test")
	_ = AddAlias(db, mergeID, "Same Name", "test")
	// Both have interactions on the same channel.
	now := time.Now()
	_ = UpsertInteraction(db, keepID, "whatsapp", 10, &now)
	_ = UpsertInteraction(db, mergeID, "whatsapp", 5, &now)
	// mergeID also has a unique email.
	_ = UpsertIdentity(db, &Identity{ContactID: mergeID, Source: "gmail", IdentityType: "email", IdentityValue: "m@test.com"})

	if err := MergeContacts(db, keepID, mergeID); err != nil {
		t.Fatalf("merge with conflicts: %v", err)
	}

	kept, _ := GetContact(db, keepID)
	if len(kept.Identities) != 2 {
		t.Errorf("kept identities = %d, want 2 (phone + email)", len(kept.Identities))
	}
	if len(kept.Interactions) != 1 {
		t.Errorf("kept interactions = %d, want 1", len(kept.Interactions))
	}

	merged, _ := GetContact(db, mergeID)
	if merged != nil {
		t.Error("merged contact should be deleted")
	}
}

func TestSyncState(t *testing.T) {
	db := testDB(t)
	ts, cursor, err := GetSyncState(db, "whatsapp")
	if err != nil {
		t.Fatal(err)
	}
	if ts != nil || cursor != "" {
		t.Error("expected nil state for new source")
	}

	if err := SaveSyncState(db, "whatsapp", "cursor1"); err != nil {
		t.Fatal(err)
	}
	ts, cursor, err = GetSyncState(db, "whatsapp")
	if err != nil {
		t.Fatal(err)
	}
	if ts == nil {
		t.Error("expected non-nil timestamp")
	}
	if cursor != "cursor1" {
		t.Errorf("cursor = %q, want cursor1", cursor)
	}
}
