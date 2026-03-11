package contacts

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/priyanshujain/openbotkit/store"
)

func syncFromWhatsApp(contactsDB, waDB *store.DB) (*SyncResult, error) {
	result := &SyncResult{}

	rows, err := waDB.Query("SELECT jid, phone, first_name, full_name, push_name, business_name FROM whatsapp_contacts")
	if err != nil {
		return nil, fmt.Errorf("query whatsapp contacts: %w", err)
	}
	defer rows.Close()

	type waContact struct {
		jid, phone, firstName, fullName, pushName, businessName string
	}
	var contacts []waContact
	for rows.Next() {
		var jid, phone sql.NullString
		var firstName, fullName, pushName, businessName sql.NullString
		if err := rows.Scan(&jid, &phone, &firstName, &fullName, &pushName, &businessName); err != nil {
			return nil, fmt.Errorf("scan whatsapp contact: %w", err)
		}
		contacts = append(contacts, waContact{
			jid: jid.String, phone: phone.String,
			firstName: firstName.String, fullName: fullName.String,
			pushName: pushName.String, businessName: businessName.String,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, wc := range contacts {
		phone := ExtractPhoneFromJID(wc.jid)
		if phone == "" {
			continue
		}

		existing, err := FindContactByIdentity(contactsDB, "phone", phone)
		if err != nil {
			slog.Error("contacts: find by phone", "phone", phone, "error", err)
			result.Errors++
			continue
		}

		var contactID int64
		if existing != nil {
			contactID = existing.ID
			result.Linked++
		} else {
			displayName := bestName(wc.fullName, wc.pushName, wc.firstName, wc.businessName, wc.phone)
			contactID, err = CreateContact(contactsDB, displayName)
			if err != nil {
				slog.Error("contacts: create from whatsapp", "jid", wc.jid, "error", err)
				result.Errors++
				continue
			}
			result.Created++
		}

		if err := UpsertIdentity(contactsDB, &Identity{
			ContactID: contactID, Source: "whatsapp", IdentityType: "wa_jid",
			IdentityValue: wc.jid, DisplayName: wc.fullName, RawValue: wc.jid,
		}); err != nil {
			result.Errors++
			continue
		}
		if err := UpsertIdentity(contactsDB, &Identity{
			ContactID: contactID, Source: "whatsapp", IdentityType: "phone",
			IdentityValue: phone, RawValue: wc.phone,
		}); err != nil {
			result.Errors++
			continue
		}

		for _, name := range []string{wc.fullName, wc.pushName, wc.firstName, wc.businessName} {
			_ = AddAlias(contactsDB, contactID, name, "whatsapp")
		}

		if err := syncWhatsAppInteractions(contactsDB, waDB, contactID, wc.jid); err != nil {
			slog.Error("contacts: whatsapp interactions", "jid", wc.jid, "error", err)
		}
	}

	return result, nil
}

func syncWhatsAppInteractions(contactsDB, waDB *store.DB, contactID int64, jid string) error {
	var count int
	var lastAtRaw sql.NullString
	err := waDB.QueryRow(
		waDB.Rebind("SELECT COUNT(*), MAX(timestamp) FROM whatsapp_messages WHERE sender_jid = ?"),
		jid,
	).Scan(&count, &lastAtRaw)
	if err != nil {
		return err
	}
	if count == 0 {
		return nil
	}
	var t *time.Time
	if lastAtRaw.Valid && lastAtRaw.String != "" {
		if parsed, err := time.Parse("2006-01-02 15:04:05", lastAtRaw.String); err == nil {
			t = &parsed
		} else if parsed, err := time.Parse(time.RFC3339, lastAtRaw.String); err == nil {
			t = &parsed
		}
	}
	return UpsertInteraction(contactsDB, contactID, "whatsapp", count, t)
}

func bestName(names ...string) string {
	for _, n := range names {
		if n != "" {
			return n
		}
	}
	return "Unknown"
}
