package contacts

import (
	"fmt"
	"log/slog"
	"runtime"
	"strings"

	"github.com/73ai/openbotkit/internal/obkmacos"
	"github.com/73ai/openbotkit/store"
)

func syncFromAppleContacts(contactsDB *store.DB) (*SyncResult, error) {
	if runtime.GOOS != "darwin" {
		return &SyncResult{}, nil
	}

	people, err := fetchAppleContacts()
	if err != nil {
		return nil, fmt.Errorf("fetch apple contacts: %w", err)
	}

	result := &SyncResult{}
	for _, p := range people {
		if err := importApplePerson(contactsDB, p, result); err != nil {
			slog.Error("contacts: import apple contact", "name", p.firstName+" "+p.lastName, "error", err)
			result.Errors++
		}
	}
	return result, nil
}

type applePerson struct {
	firstName string
	lastName  string
	nickname  string
	phones    []string
	emails    []string
}

func fetchAppleContacts() ([]applePerson, error) {
	contacts, err := obkmacos.FetchContacts()
	if err != nil {
		return nil, err
	}
	return convertContacts(contacts), nil
}

func convertContacts(contacts []obkmacos.Contact) []applePerson {
	var people []applePerson
	for _, c := range contacts {
		p := applePerson{
			firstName: c.FirstName,
			lastName:  c.LastName,
			nickname:  c.Nickname,
			phones:    c.Phones,
			emails:    c.Emails,
		}
		if p.firstName == "" && p.lastName == "" && len(p.phones) == 0 && len(p.emails) == 0 {
			continue
		}
		people = append(people, p)
	}
	return people
}

func CheckAppleContactsPermission() error {
	// FetchContacts triggers CNContactStore.requestAccess which shows the
	// macOS permission dialog for first-time users. CheckPermissions only
	// reads status without requesting, so it can't be used here.
	_, err := obkmacos.FetchContacts()
	return err
}

func importApplePerson(db *store.DB, p applePerson, result *SyncResult) error {
	var contactID int64
	var found bool

	for _, ph := range p.phones {
		normalized := NormalizePhone(ph)
		if normalized == "" {
			continue
		}
		existing, err := FindContactByIdentity(db, "phone", normalized)
		if err != nil {
			return err
		}
		if existing != nil {
			contactID = existing.ID
			found = true
			break
		}
	}

	if !found {
		for _, em := range p.emails {
			normalized := NormalizeEmail(em)
			if normalized == "" {
				continue
			}
			existing, err := FindContactByIdentity(db, "email", normalized)
			if err != nil {
				return err
			}
			if existing != nil {
				contactID = existing.ID
				found = true
				break
			}
		}
	}

	if found {
		result.Linked++
	} else {
		displayName := bestName(strings.TrimSpace(p.firstName+" "+p.lastName), p.firstName, p.lastName, p.nickname)
		var err error
		contactID, err = CreateContact(db, displayName)
		if err != nil {
			return err
		}
		result.Created++
	}

	for _, ph := range p.phones {
		normalized := NormalizePhone(ph)
		if normalized == "" {
			continue
		}
		if err := UpsertIdentity(db, &Identity{
			ContactID: contactID, Source: "applecontacts", IdentityType: "phone",
			IdentityValue: normalized, RawValue: ph,
		}); err != nil {
			slog.Warn("contacts: apple upsert phone identity", "phone", normalized, "error", err)
		}
	}
	for _, em := range p.emails {
		normalized := NormalizeEmail(em)
		if normalized == "" {
			continue
		}
		if err := UpsertIdentity(db, &Identity{
			ContactID: contactID, Source: "applecontacts", IdentityType: "email",
			IdentityValue: normalized, RawValue: em,
		}); err != nil {
			slog.Warn("contacts: apple upsert email identity", "email", normalized, "error", err)
		}
	}

	fullName := strings.TrimSpace(p.firstName + " " + p.lastName)
	for _, name := range []string{fullName, p.firstName, p.lastName, p.nickname} {
		if err := AddAlias(db, contactID, name, "applecontacts"); err != nil {
			slog.Warn("contacts: apple add alias", "alias", name, "error", err)
		}
	}

	return nil
}
