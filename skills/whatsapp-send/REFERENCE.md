## Command

```bash
obk whatsapp messages send --to <jid> --text "<message>"
```

## Finding the recipient

Always resolve the recipient before sending. Use unified contacts first, then fall back to WhatsApp-specific queries.

```bash
# Step 1: Search unified contacts (preferred — cross-source, ranked by frequency)
obk contacts search "David"

# Step 2: Use the WhatsApp JID from the results to send
obk whatsapp messages send --to "<jid>" --text "..."

# Fallback: search WhatsApp contacts directly
obk db whatsapp "SELECT jid, phone, full_name, push_name FROM whatsapp_contacts WHERE LOWER(full_name) LIKE '%name%' OR LOWER(push_name) LIKE '%name%' OR LOWER(first_name) LIKE '%name%';"

# Fallback: search chats if not found in contacts
obk db whatsapp "SELECT jid, name FROM whatsapp_chats WHERE LOWER(name) LIKE '%name%';"
```

JID formats:
- Individual: `<phone>@s.whatsapp.net` (e.g. `919876543210@s.whatsapp.net`)
- Group: `<id>@g.us`

## Confirmation rules

- If the user's intent is clear and only ONE contact matches → send immediately, no need to confirm
- If MULTIPLE contacts match → show the matches and ask the user to pick
- If NO contacts match → tell the user and ask for clarification
- Only confirm content if the user's message is ambiguous

## Example

```bash
# Send a message to a contact
obk whatsapp messages send --to "919876543210@s.whatsapp.net" --text "Hello!"

# Send a message to a group
obk whatsapp messages send --to "120363001234567890@g.us" --text "Hey everyone"
```

## Notes

- Requires an authenticated WhatsApp session (`obk whatsapp auth login`)
- The sent message is saved to the local database automatically
