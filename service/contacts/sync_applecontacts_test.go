//go:build darwin && integration

package contacts

import "testing"

func TestCheckAppleContactsPermission_Integration(t *testing.T) {
	if err := CheckAppleContactsPermission(); err != nil {
		t.Logf("permission check failed (may need to grant access): %v", err)
	}
}

func TestFetchAppleContacts_Integration(t *testing.T) {
	people, err := fetchAppleContacts()
	if err != nil {
		t.Fatalf("fetchAppleContacts: %v", err)
	}

	t.Logf("fetched %d contacts from Apple Contacts", len(people))

	for i, p := range people {
		if i >= 5 {
			break
		}
		t.Logf("  [%d] first=%q last=%q phones=%d emails=%d",
			i, p.firstName, p.lastName, len(p.phones), len(p.emails))
	}
}

func TestSyncFromAppleContacts_Integration(t *testing.T) {
	db := testDB(t)
	result, err := syncFromAppleContacts(db)
	if err != nil {
		t.Fatalf("syncFromAppleContacts: %v", err)
	}

	t.Logf("sync result: created=%d linked=%d errors=%d", result.Created, result.Linked, result.Errors)

	count, _ := CountContacts(db)
	t.Logf("total contacts in DB: %d", count)

	if count == 0 && result.Errors == 0 {
		t.Log("warning: no contacts synced (Contacts app may be empty or permissions not granted)")
	}

	contacts, _ := ListContacts(db, 5, 0)
	for _, c := range contacts {
		full, _ := GetContact(db, c.ID)
		t.Logf("  %s: %d identities, %d aliases",
			full.DisplayName, len(full.Identities), len(full.Aliases))
	}
}
