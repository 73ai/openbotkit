package contacts

import (
	"fmt"
	"log/slog"
	"os/exec"
	"runtime"
	"strings"

	"github.com/priyanshujain/openbotkit/store"
)

const acFieldSep = "%%FIELD%%"
const acRecordSep = "%%RECORD%%"

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
	script := fmt.Sprintf(`
tell application "Contacts"
	set output to ""
	repeat with p in every person
		set fn to first name of p
		if fn is missing value then set fn to ""
		set ln to last name of p
		if ln is missing value then set ln to ""
		set nn to nickname of p
		if nn is missing value then set nn to ""

		set phoneList to ""
		repeat with ph in phones of p
			set phoneList to phoneList & value of ph & ","
		end repeat

		set emailList to ""
		repeat with em in emails of p
			set emailList to emailList & value of em & ","
		end repeat

		set output to output & fn & "%s" & ln & "%s" & nn & "%s" & phoneList & "%s" & emailList & "%s"
	end repeat
	return output
end tell`, acFieldSep, acFieldSep, acFieldSep, acFieldSep, acRecordSep)

	cmd := exec.Command("osascript", "-e", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("osascript: %s: %w", strings.TrimSpace(string(out)), err)
	}

	output := strings.TrimSpace(string(out))
	if output == "" {
		return nil, nil
	}

	records := strings.Split(output, acRecordSep)
	var people []applePerson
	for _, rec := range records {
		rec = strings.TrimSpace(rec)
		if rec == "" {
			continue
		}
		fields := strings.Split(rec, acFieldSep)
		if len(fields) < 5 {
			continue
		}
		p := applePerson{
			firstName: strings.TrimSpace(fields[0]),
			lastName:  strings.TrimSpace(fields[1]),
			nickname:  strings.TrimSpace(fields[2]),
		}
		for _, ph := range strings.Split(fields[3], ",") {
			ph = strings.TrimSpace(ph)
			if ph != "" {
				p.phones = append(p.phones, ph)
			}
		}
		for _, em := range strings.Split(fields[4], ",") {
			em = strings.TrimSpace(em)
			if em != "" {
				p.emails = append(p.emails, em)
			}
		}
		if p.firstName == "" && p.lastName == "" && len(p.phones) == 0 && len(p.emails) == 0 {
			continue
		}
		people = append(people, p)
	}
	return people, nil
}

func importApplePerson(db *store.DB, p applePerson, result *SyncResult) error {
	// Try to find existing contact by any phone or email.
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
		_ = UpsertIdentity(db, &Identity{
			ContactID: contactID, Source: "applecontacts", IdentityType: "phone",
			IdentityValue: normalized, RawValue: ph,
		})
	}
	for _, em := range p.emails {
		normalized := NormalizeEmail(em)
		if normalized == "" {
			continue
		}
		_ = UpsertIdentity(db, &Identity{
			ContactID: contactID, Source: "applecontacts", IdentityType: "email",
			IdentityValue: normalized, RawValue: em,
		})
	}

	fullName := strings.TrimSpace(p.firstName + " " + p.lastName)
	for _, name := range []string{fullName, p.firstName, p.lastName, p.nickname} {
		_ = AddAlias(db, contactID, name, "applecontacts")
	}

	return nil
}
