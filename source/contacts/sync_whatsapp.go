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
		var c waContact
		if err := rows.Scan(&c.jid, &c.phone, &c.firstName, &c.fullName, &c.pushName, &c.businessName); err != nil {
			return nil, fmt.Errorf("scan whatsapp contact: %w", err)
		}
		contacts = append(contacts, c)
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
	var lastAt sql.NullTime
	err := waDB.QueryRow(
		waDB.Rebind("SELECT COUNT(*), MAX(timestamp) FROM whatsapp_messages WHERE sender_jid = ? OR chat_jid = ?"),
		jid, jid,
	).Scan(&count, &lastAt)
	if err != nil {
		return err
	}
	if count == 0 {
		return nil
	}
	var t *time.Time
	if lastAt.Valid {
		t = &lastAt.Time
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
