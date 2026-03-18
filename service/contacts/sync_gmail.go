package contacts

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/73ai/openbotkit/store"
)

func syncFromGmail(contactsDB, gmailDB *store.DB) (*SyncResult, error) {
	result := &SyncResult{}

	rows, err := gmailDB.Query(`
		SELECT addr, SUM(cnt) as total, MAX(last_date) as last_date FROM (
			SELECT from_addr as addr, COUNT(*) as cnt, MAX(date) as last_date FROM emails WHERE from_addr != '' GROUP BY from_addr
			UNION ALL
			SELECT to_addr as addr, COUNT(*) as cnt, MAX(date) as last_date FROM emails WHERE to_addr != '' GROUP BY to_addr
		) GROUP BY addr`)
	if err != nil {
		return nil, fmt.Errorf("query gmail addresses: %w", err)
	}
	defer rows.Close()

	type gmailAddr struct {
		raw       string
		count     int
		lastAtRaw sql.NullString
	}
	var addrs []gmailAddr
	for rows.Next() {
		var a gmailAddr
		if err := rows.Scan(&a.raw, &a.count, &a.lastAtRaw); err != nil {
			return nil, fmt.Errorf("scan gmail addr: %w", err)
		}
		addrs = append(addrs, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, ga := range addrs {
		name, email := ParseEmailAddr(ga.raw)
		if email == "" {
			continue
		}

		existing, err := FindContactByIdentity(contactsDB, "email", email)
		if err != nil {
			slog.Error("contacts: find by email", "email", email, "error", err)
			result.Errors++
			continue
		}

		var contactID int64
		if existing != nil {
			contactID = existing.ID
			result.Linked++
		} else {
			displayName := name
			if displayName == "" {
				displayName = email
			}
			contactID, err = CreateContact(contactsDB, displayName)
			if err != nil {
				slog.Error("contacts: create from gmail", "email", email, "error", err)
				result.Errors++
				continue
			}
			result.Created++
		}

		if err := UpsertIdentity(contactsDB, &Identity{
			ContactID: contactID, Source: "gmail", IdentityType: "email",
			IdentityValue: email, DisplayName: name, RawValue: ga.raw,
		}); err != nil {
			result.Errors++
			continue
		}

		if name != "" {
			_ = AddAlias(contactsDB, contactID, name, "gmail")
		}

		var t *time.Time
		if ga.lastAtRaw.Valid && ga.lastAtRaw.String != "" {
			if parsed, err := time.Parse("2006-01-02 15:04:05", ga.lastAtRaw.String); err == nil {
				t = &parsed
			} else if parsed, err := time.Parse(time.RFC3339, ga.lastAtRaw.String); err == nil {
				t = &parsed
			}
		}
		if err := UpsertInteraction(contactsDB, contactID, "gmail", ga.count, t); err != nil {
			slog.Error("contacts: gmail interaction", "email", email, "error", err)
		}
	}

	return result, nil
}
