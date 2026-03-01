# OpenBotKit

A toolkit for building AI personal assistants through data source integrations.

OpenBotKit (`obk`) serves dual purposes: a **CLI tool** for syncing and querying personal data, and a **Go library** that agent developers can import to build AI assistants.

## Features

- **Gmail** — Sync, search, read, send emails, and create drafts across multiple accounts
- **WhatsApp** — Sync messages, search conversations, send messages to contacts and groups
- **Memory** — Capture and recall past assistant conversations
- **Background daemon** — Continuous syncing via launchd/systemd
- **AI assistant integration** — Claude Code skills for natural-language access to all data

## Install

```bash
go install github.com/priyanshujain/openbotkit@latest
```

Or build from source:

```bash
git clone https://github.com/priyanshujain/openbotkit.git
cd openbotkit
go build -o obk .
```

## Quick Start

```bash
# Initialize configuration
obk config init

# --- Gmail ---
cp credentials.json ~/.obk/gmail/credentials.json
obk gmail auth login
obk gmail sync
obk gmail emails list
obk gmail emails search "invoice"

# --- WhatsApp ---
obk whatsapp auth login    # scan QR code
obk whatsapp sync
obk whatsapp chats list
obk whatsapp messages search "meeting"

# Check status of all sources
obk status
```

## CLI Commands

### General

```
obk version                          # Print version
obk status                           # All sources: connected?, items, last sync

obk config init                      # Create default config at ~/.obk/config.yaml
obk config show                      # Print resolved config
obk config set <key> <value>         # Set a config value
obk config path                      # Print config directory
```

### Gmail

```
obk gmail auth login                 # OAuth2 browser flow
obk gmail auth logout [--account]    # Remove stored tokens
obk gmail auth status                # Show connected accounts

obk gmail sync                       # Incremental sync
    [--account EMAIL]                # Filter to one account
    [--full]                         # Re-fetch everything
    [--after DATE]                   # Only emails after this date
    [--days N]                       # Days to sync (default 7, 0 for all)
    [--download-attachments]         # Save attachments to disk

obk gmail fetch                      # On-demand fetch from Gmail API
    --account EMAIL                  # Account email (required)
    [--after DATE]                   # Fetch emails after date (YYYY/MM/DD)
    [--before DATE]                  # Fetch emails before date (YYYY/MM/DD)
    [--query QUERY]                  # Raw Gmail search query
    [--download-attachments]         # Save attachments to disk
    [--json]                         # Output as JSON

obk gmail emails list                # Paginated list of stored emails
    [--account EMAIL] [--from ADDR]
    [--subject TEXT] [--after DATE]
    [--before DATE] [--limit N]
    [--json]

obk gmail emails get <message-id>    # Full email details
    [--json]

obk gmail emails search <query>      # Full-text search
    [--limit N] [--json]

obk gmail send                       # Send an email
    --to ADDR [--to ADDR ...]        # Recipients (required)
    --subject TEXT                   # Subject (required)
    --body TEXT                      # Body (required)
    [--cc ADDR] [--bcc ADDR]
    [--account EMAIL]

obk gmail drafts create              # Create a draft email
    --to ADDR [--to ADDR ...]        # Recipients (required)
    [--subject TEXT] [--body TEXT]
    [--cc ADDR] [--bcc ADDR]
    [--account EMAIL]
```

### WhatsApp

```
obk whatsapp auth login              # QR code authentication
obk whatsapp auth logout             # Remove session

obk whatsapp sync                    # Sync messages

obk whatsapp chats list              # List all synced chats
    [--json]

obk whatsapp messages list           # List stored messages
    [--chat JID]                     # Filter by chat
    [--after DATE] [--before DATE]
    [--limit N] [--json]

obk whatsapp messages search <query> # Full-text search
    [--limit N] [--json]

obk whatsapp messages send           # Send a message
    --to JID                         # Recipient JID (required)
    --text MESSAGE                   # Message text (required)
```

### Memory

```
obk memory capture                   # Capture conversation from stdin (JSON)
```

### Daemon & Service

```
obk daemon                           # Run background daemon
    [--mode standalone|worker]       # Daemon mode (default: standalone)

obk service install                  # Install as system service (launchd/systemd)
obk service uninstall                # Uninstall system service
obk service status                   # Check service status
```

## AI Assistant

OpenBotKit includes a pre-configured Claude Code assistant setup in the `assistant/` directory. Copy it to use as your personal assistant with natural-language access to your data.

Available skills:
- **email-read** — Search emails, check inbox, find messages
- **email-send** — Send emails and create drafts
- **whatsapp-read** — Search WhatsApp messages and chats
- **whatsapp-send** — Send WhatsApp messages to contacts and groups
- **memory-read** — Recall past conversations and discussion history

See `assistant/` for setup instructions.

## Library Usage

```go
import (
    "github.com/priyanshujain/openbotkit/source/gmail"
    "github.com/priyanshujain/openbotkit/store"
)

// Open database
db, _ := store.Open(store.SQLiteConfig("gmail.db"))
gmail.Migrate(db)

// Create Gmail source
g := gmail.New(gmail.Config{
    CredentialsFile: "credentials.json",
    TokenDBPath:     "tokens.db",
})

// Sync emails
result, _ := g.Sync(ctx, db, gmail.SyncOptions{Full: false})

// Query stored emails
emails, _ := gmail.ListEmails(db, gmail.ListOptions{
    From:  "someone@example.com",
    Limit: 10,
})
```

## Configuration

Config lives at `~/.obk/config.yaml` (override with `OBK_CONFIG_DIR`):

```yaml
gmail:
  credentials_file: ~/.obk/gmail/credentials.json
  download_attachments: false
  storage:
    driver: sqlite    # or "postgres"
    dsn: ""           # postgres DSN; sqlite path auto-derived

whatsapp:
  storage:
    driver: sqlite

memory:
  storage:
    driver: sqlite
```

## Data Directory

```
~/.obk/
├── config.yaml
├── gmail/
│   ├── credentials.json    # Google OAuth client creds (user provides)
│   ├── tokens.db           # OAuth tokens (always local SQLite)
│   ├── data.db             # Email data (when using SQLite)
│   └── attachments/        # Downloaded attachments
├── whatsapp/
│   ├── session.db          # WhatsApp session data
│   └── data.db             # Message data
└── memory/
    └── data.db             # Conversation history
```

## Prerequisites

- Go 1.25+
- Gmail API credentials ([Google Cloud Console](https://console.cloud.google.com/apis/credentials))
