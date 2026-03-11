## Commands

### Search contacts by name

```bash
obk contacts search "David"
obk contacts search "David" --json
obk contacts search "David" --limit 5
```

Returns matching contacts ranked by: exact match > prefix match > substring, then by interaction frequency.

### Get full contact details

```bash
obk contacts get <id>
obk contacts get <id> --json
```

Shows all identities (phone, email, WhatsApp JID), aliases, and interaction counts.

### List all contacts

```bash
obk contacts list
obk contacts list --limit 20
obk contacts list --json
```

### Trigger manual sync

```bash
obk contacts sync
```

## Raw SQL queries

For advanced queries, use `obk db contacts`:

```bash
# Search by identity value (phone or email)
obk db contacts "SELECT c.id, c.display_name FROM contacts c JOIN contact_identities ci ON ci.contact_id = c.id WHERE ci.identity_value = '+919876543210';"

# Find contacts with most interactions
obk db contacts "SELECT c.id, c.display_name, SUM(ci.message_count) as total FROM contacts c JOIN contact_interactions ci ON ci.contact_id = c.id GROUP BY c.id ORDER BY total DESC LIMIT 10;"

# Find all identities for a contact
obk db contacts "SELECT source, identity_type, identity_value FROM contact_identities WHERE contact_id = 1;"

# Search by alias
obk db contacts "SELECT DISTINCT c.id, c.display_name FROM contacts c JOIN contact_aliases a ON a.contact_id = c.id WHERE a.alias_lower LIKE '%david%';"
```

## Schema

- `contacts` — id, display_name, created_at, updated_at
- `contact_identities` — contact_id, source, identity_type, identity_value, display_name, raw_value
- `contact_aliases` — contact_id, alias, alias_lower, source
- `contact_interactions` — contact_id, channel, message_count, last_interaction_at
- `contact_sync_state` — source, last_synced_at, last_cursor

## Notes

- Contacts are aggregated from WhatsApp, Gmail, iMessage, and Apple Contacts
- Cross-source dedup uses phone number and email as merge keys
- Names alone do NOT trigger auto-merge — two "David" with different phones stay separate
- The daemon syncs contacts every 5 minutes; use `obk contacts sync` for immediate refresh
