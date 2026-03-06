# OpenBotKit

A DIY kit for building your own AI assistant. Runs on your machine, talks to your data, answers to you.

## Why

Most AI assistants are meals. Pre-cooked, opinionated, served through someone else's kitchen. You eat what you're given. Your data passes through their servers. Their defaults become your constraints. When something breaks or doesn't fit, you can't open the hood.

OpenBotKit is a meal kit. You get the raw ingredients (data connectors, a sync engine, a local database, a CLI, and assistant scaffolding) and you put the assistant together yourself. You pick which data sources to connect. You pick what the assistant can see and do. You control the recipe.

### What this means in practice

- **Your data stays on your machine.** Emails, messages, and notes sync into local SQLite databases under `~/.obk/`. Nothing leaves your device unless you explicitly send it.
- **No intermediary.** `obk` connects directly to Gmail's API, WhatsApp's protocol, and Apple Notes on your Mac. No relay server, no cloud sync layer, no third-party backend.
- **You see every action.** The assistant runs through Claude Code. Every database query, every message sent, every command executed, you see it and can approve it. No autonomous loops running behind your back.
- **You own the code.** It's a Go binary and a set of SQLite files. Fork it, extend it, rip out what you don't need.

## What's in the kit

| Component | What it does |
|---|---|
| **Sources** | Connectors for Gmail, WhatsApp, Apple Notes, and conversation memory |
| **Sync engine** | Background daemon (launchd/systemd) keeps your local data fresh |
| **CLI** (`obk`) | Search, read, and send across all sources from the terminal |
| **Assistant scaffolding** | Pre-configured Claude Code setup with skills for natural-language access |
| **Go library** | Import `openbotkit/source/*` to build your own tools and integrations |

## Install

```bash
go install github.com/priyanshujain/openbotkit@latest
```

Or build from source:

```bash
git clone https://github.com/priyanshujain/openbotkit.git
cd openbotkit && make install
```

## Quick Start

```bash
# Guided setup: pick your sources, authenticate, run first sync
obk setup

# Or configure manually:
obk config init
obk gmail auth login          # OAuth2 browser flow
obk gmail sync
obk whatsapp auth login       # scan QR code
obk whatsapp sync

# Check what's connected
obk status
```

## Building Your Assistant

OpenBotKit ships an `assistant/` directory, a ready-to-use Claude Code workspace with skills wired to your synced data.

```bash
# Symlink outside the repo (avoids loading dev-only CLAUDE.md)
ln -s /path/to/openbotkit/assistant ~/assistant
cd ~/assistant && claude
```

From there you can ask things like:

- *"Do I have any unread emails from Stripe?"*
- *"Tell David I'll be 10 minutes late"* (sends via WhatsApp)
- *"Draft a reply to the invoice email from yesterday"*
- *"What did we discuss about the API redesign last week?"*
- *"Find my notes about the Berlin trip"*

Each skill is just a plain text file with SQL patterns and CLI commands. You can read them, change them, or write your own. No magic. See [`assistant/`](assistant/) for setup details.

## Library Usage

`obk` is also a Go library. Import the source packages to build your own tools on top.

```go
import (
    "github.com/priyanshujain/openbotkit/source/gmail"
    "github.com/priyanshujain/openbotkit/store"
)

db, _ := store.Open(store.SQLiteConfig("gmail.db"))
gmail.Migrate(db)

g := gmail.New(gmail.Config{
    CredentialsFile: "credentials.json",
    TokenDBPath:     "tokens.db",
})

result, _ := g.Sync(ctx, db, gmail.SyncOptions{Full: false})

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

applenotes:
  storage:
    driver: sqlite
```

### Data directory

```
~/.obk/
├── config.yaml
├── gmail/
│   ├── credentials.json    # Google OAuth client creds
│   ├── tokens.db           # OAuth tokens
│   ├── data.db             # Synced emails
│   └── attachments/        # Downloaded attachments
├── whatsapp/
│   ├── session.db          # WhatsApp session
│   └── data.db             # Synced messages
├── applenotes/
│   └── data.db             # Synced notes
└── memory/
    └── data.db             # Conversation history
```

<details>
<summary><strong>CLI Reference</strong></summary>

### General

```
obk version                          # Print version
obk setup                           # Guided setup wizard
obk status                          # All sources: connected?, items, last sync

obk config init                     # Create default config at ~/.obk/config.yaml
obk config show                     # Print resolved config
obk config set <key> <value>        # Set a config value
obk config path                     # Print config directory
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

### Apple Notes

```
obk applenotes sync                  # Sync notes from Apple Notes
obk applenotes notes list            # List synced notes
    [--folder NAME] [--limit N] [--json]
obk applenotes notes search <query>  # Full-text search
    [--limit N] [--json]
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

</details>

## Prerequisites

- Go 1.25+
- Gmail requires API credentials from [Google Cloud Console](https://console.cloud.google.com/apis/credentials)
- WhatsApp requires scanning a QR code to link your phone
- Apple Notes requires macOS (uses AppleScript, no special permissions needed)

## License

MIT
