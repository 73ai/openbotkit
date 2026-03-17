package contacts

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/73ai/openbotkit/store"
)

func CreateContact(db *store.DB, displayName string) (int64, error) {
	result, err := db.Exec(
		db.Rebind("INSERT INTO contacts (display_name) VALUES (?)"),
		displayName,
	)
	if err != nil {
		return 0, fmt.Errorf("create contact: %w", err)
	}
	return result.LastInsertId()
}

func GetContact(db *store.DB, id int64) (*Contact, error) {
	var c Contact
	err := db.QueryRow(
		db.Rebind("SELECT id, display_name, created_at, updated_at FROM contacts WHERE id = ?"),
		id,
	).Scan(&c.ID, &c.DisplayName, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get contact: %w", err)
	}

	identities, err := listIdentities(db, c.ID)
	if err != nil {
		return nil, err
	}
	c.Identities = identities

	aliases, err := listAliases(db, c.ID)
	if err != nil {
		return nil, err
	}
	c.Aliases = aliases

	interactions, err := listInteractions(db, c.ID)
	if err != nil {
		return nil, err
	}
	c.Interactions = interactions

	return &c, nil
}

func DeleteContact(db *store.DB, id int64) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("delete contact: begin tx: %w", err)
	}
	defer tx.Rollback()

	for _, table := range []string{"contact_interactions", "contact_aliases", "contact_identities"} {
		if _, err := tx.Exec(db.Rebind(fmt.Sprintf("DELETE FROM %s WHERE contact_id = ?", table)), id); err != nil {
			return fmt.Errorf("delete %s: %w", table, err)
		}
	}
	if _, err := tx.Exec(db.Rebind("DELETE FROM contacts WHERE id = ?"), id); err != nil {
		return fmt.Errorf("delete contact: %w", err)
	}
	return tx.Commit()
}

func ListContacts(db *store.DB, limit, offset int) ([]Contact, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := db.Query(
		db.Rebind("SELECT id, display_name, created_at, updated_at FROM contacts ORDER BY display_name LIMIT ? OFFSET ?"),
		limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list contacts: %w", err)
	}
	defer rows.Close()

	contacts := []Contact{}
	for rows.Next() {
		var c Contact
		if err := rows.Scan(&c.ID, &c.DisplayName, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan contact: %w", err)
		}
		contacts = append(contacts, c)
	}
	return contacts, rows.Err()
}

func CountContacts(db *store.DB) (int64, error) {
	var count int64
	err := db.QueryRow("SELECT COUNT(*) FROM contacts").Scan(&count)
	return count, err
}

func UpsertIdentity(db *store.DB, ident *Identity) error {
	_, err := db.Exec(
		db.Rebind(`INSERT INTO contact_identities (contact_id, source, identity_type, identity_value, display_name, raw_value)
			VALUES (?, ?, ?, ?, ?, ?)
			ON CONFLICT(source, identity_type, identity_value) DO UPDATE SET
				contact_id = excluded.contact_id,
				display_name = CASE WHEN excluded.display_name != '' THEN excluded.display_name ELSE contact_identities.display_name END,
				raw_value = CASE WHEN excluded.raw_value != '' THEN excluded.raw_value ELSE contact_identities.raw_value END,
				updated_at = CURRENT_TIMESTAMP`),
		ident.ContactID, ident.Source, ident.IdentityType, ident.IdentityValue, ident.DisplayName, ident.RawValue,
	)
	if err != nil {
		return fmt.Errorf("upsert identity: %w", err)
	}
	return nil
}

func FindContactByIdentity(db *store.DB, identityType, identityValue string) (*Contact, error) {
	var contactID int64
	err := db.QueryRow(
		db.Rebind("SELECT contact_id FROM contact_identities WHERE identity_type = ? AND identity_value = ? LIMIT 1"),
		identityType, identityValue,
	).Scan(&contactID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find contact by identity: %w", err)
	}
	return GetContact(db, contactID)
}

func AddAlias(db *store.DB, contactID int64, alias, source string) error {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return nil
	}
	_, err := db.Exec(
		db.Rebind(`INSERT INTO contact_aliases (contact_id, alias, alias_lower, source)
			VALUES (?, ?, ?, ?)
			ON CONFLICT(contact_id, alias_lower) DO NOTHING`),
		contactID, alias, strings.ToLower(alias), source,
	)
	if err != nil {
		return fmt.Errorf("add alias: %w", err)
	}
	return nil
}

func UpsertInteraction(db *store.DB, contactID int64, channel string, count int, lastAt *time.Time) error {
	_, err := db.Exec(
		db.Rebind(`INSERT INTO contact_interactions (contact_id, channel, message_count, last_interaction_at)
			VALUES (?, ?, ?, ?)
			ON CONFLICT(contact_id, channel) DO UPDATE SET
				message_count = excluded.message_count,
				last_interaction_at = excluded.last_interaction_at`),
		contactID, channel, count, lastAt,
	)
	if err != nil {
		return fmt.Errorf("upsert interaction: %w", err)
	}
	return nil
}

func MergeContacts(db *store.DB, keepID, mergeID int64) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("merge contacts: begin tx: %w", err)
	}
	defer tx.Rollback()

	// Delete duplicate identities that would violate UNIQUE(source, identity_type, identity_value).
	if _, err := tx.Exec(db.Rebind(`DELETE FROM contact_identities WHERE contact_id = ? AND EXISTS (
		SELECT 1 FROM contact_identities k WHERE k.contact_id = ?
		AND k.source = contact_identities.source
		AND k.identity_type = contact_identities.identity_type
		AND k.identity_value = contact_identities.identity_value)`), mergeID, keepID); err != nil {
		return fmt.Errorf("merge: dedup identities: %w", err)
	}
	// Delete duplicate aliases that would violate UNIQUE(contact_id, alias_lower).
	if _, err := tx.Exec(db.Rebind(`DELETE FROM contact_aliases WHERE contact_id = ? AND alias_lower IN (
		SELECT alias_lower FROM contact_aliases WHERE contact_id = ?)`), mergeID, keepID); err != nil {
		return fmt.Errorf("merge: dedup aliases: %w", err)
	}
	// Delete duplicate interactions that would violate UNIQUE(contact_id, channel).
	if _, err := tx.Exec(db.Rebind(`DELETE FROM contact_interactions WHERE contact_id = ? AND channel IN (
		SELECT channel FROM contact_interactions WHERE contact_id = ?)`), mergeID, keepID); err != nil {
		return fmt.Errorf("merge: dedup interactions: %w", err)
	}

	// Move remaining rows to keepID.
	for _, q := range []string{
		"UPDATE contact_identities SET contact_id = ? WHERE contact_id = ?",
		"UPDATE contact_aliases SET contact_id = ? WHERE contact_id = ?",
		"UPDATE contact_interactions SET contact_id = ? WHERE contact_id = ?",
	} {
		if _, err := tx.Exec(db.Rebind(q), keepID, mergeID); err != nil {
			return fmt.Errorf("merge contacts: %w", err)
		}
	}

	// Delete the merged contact and its remaining child rows.
	for _, table := range []string{"contact_interactions", "contact_aliases", "contact_identities"} {
		if _, err := tx.Exec(db.Rebind(fmt.Sprintf("DELETE FROM %s WHERE contact_id = ?", table)), mergeID); err != nil {
			return fmt.Errorf("merge: delete %s: %w", table, err)
		}
	}
	if _, err := tx.Exec(db.Rebind("DELETE FROM contacts WHERE id = ?"), mergeID); err != nil {
		return fmt.Errorf("merge: delete contact: %w", err)
	}

	return tx.Commit()
}

func GetSyncState(db *store.DB, source string) (*time.Time, string, error) {
	var lastSynced sql.NullTime
	var cursor sql.NullString
	err := db.QueryRow(
		db.Rebind("SELECT last_synced_at, last_cursor FROM contact_sync_state WHERE source = ?"),
		source,
	).Scan(&lastSynced, &cursor)
	if err == sql.ErrNoRows {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", fmt.Errorf("get sync state: %w", err)
	}
	var t *time.Time
	if lastSynced.Valid {
		t = &lastSynced.Time
	}
	return t, cursor.String, nil
}

func SaveSyncState(db *store.DB, source, cursor string) error {
	_, err := db.Exec(
		db.Rebind(`INSERT INTO contact_sync_state (source, last_synced_at, last_cursor)
			VALUES (?, CURRENT_TIMESTAMP, ?)
			ON CONFLICT(source) DO UPDATE SET
				last_synced_at = CURRENT_TIMESTAMP,
				last_cursor = excluded.last_cursor`),
		source, cursor,
	)
	if err != nil {
		return fmt.Errorf("save sync state: %w", err)
	}
	return nil
}

func listIdentities(db *store.DB, contactID int64) ([]Identity, error) {
	rows, err := db.Query(
		db.Rebind("SELECT id, contact_id, source, identity_type, identity_value, display_name, raw_value FROM contact_identities WHERE contact_id = ?"),
		contactID,
	)
	if err != nil {
		return nil, fmt.Errorf("list identities: %w", err)
	}
	defer rows.Close()

	var out []Identity
	for rows.Next() {
		var i Identity
		if err := rows.Scan(&i.ID, &i.ContactID, &i.Source, &i.IdentityType, &i.IdentityValue, &i.DisplayName, &i.RawValue); err != nil {
			return nil, fmt.Errorf("scan identity: %w", err)
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func listAliases(db *store.DB, contactID int64) ([]string, error) {
	rows, err := db.Query(
		db.Rebind("SELECT alias FROM contact_aliases WHERE contact_id = ?"),
		contactID,
	)
	if err != nil {
		return nil, fmt.Errorf("list aliases: %w", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var a string
		if err := rows.Scan(&a); err != nil {
			return nil, fmt.Errorf("scan alias: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func listInteractions(db *store.DB, contactID int64) ([]Interaction, error) {
	rows, err := db.Query(
		db.Rebind("SELECT channel, message_count, last_interaction_at FROM contact_interactions WHERE contact_id = ?"),
		contactID,
	)
	if err != nil {
		return nil, fmt.Errorf("list interactions: %w", err)
	}
	defer rows.Close()

	var out []Interaction
	for rows.Next() {
		var inter Interaction
		var lastAt sql.NullTime
		if err := rows.Scan(&inter.Channel, &inter.MessageCount, &lastAt); err != nil {
			return nil, fmt.Errorf("scan interaction: %w", err)
		}
		if lastAt.Valid {
			inter.LastInteractionAt = &lastAt.Time
		}
		out = append(out, inter)
	}
	return out, rows.Err()
}
