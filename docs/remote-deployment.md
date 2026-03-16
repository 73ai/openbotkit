# Remote Deployment

OpenBotKit supports running on a remote server as a Docker container, with Telegram as the primary interaction channel and the local `obk` CLI as a controller.

## Deployment Modes

| Mode | Where it runs | Config value |
|------|---------------|-------------|
| **Local** | Everything on your machine | `mode: local` (default) |
| **Remote** | CLI on your machine, server in Docker | `mode: remote` (local config) |
| **Server** | Inside the Docker container | `mode: server` (container config) |

## Architecture

### Remote Server (Docker container)

```
┌─────────────────────────────────────────────────┐
│  obk server (single Go binary)                  │
│                                                  │
│  HTTP API (port 8443, basic auth)               │
│  ├── POST /api/db/{source}    SQL query proxy   │
│  ├── GET  /api/memory         list memories     │
│  ├── POST /api/memory         add memory        │
│  ├── DELETE /api/memory/{id}  delete memory     │
│  ├── POST /api/memory/extract run extraction    │
│  ├── POST /api/gmail/send     send email        │
│  ├── POST /api/gmail/draft    create draft      │
│  ├── POST /api/gmail/sync     trigger sync      │
│  ├── POST /api/whatsapp/send  send message      │
│  ├── POST /api/applenotes/push receive notes    │
│  ├── GET  /api/health         health check      │
│  ├── GET  /auth/whatsapp      QR code web UI    │
│  └── GET  /auth/whatsapp/api/qr  QR polling     │
│                                                  │
│  Telegram bot (long polling)                    │
│  ├── Owner-only message filtering               │
│  ├── Agent sessions with 15-min timeout         │
│  ├── Conversation history saved to DB           │
│  └── Async memory extraction after sessions     │
│                                                  │
│  Data sync daemon                               │
│  ├── WhatsApp real-time sync                    │
│  └── Gmail sync                                 │
│                                                  │
│  Data (volume: /data)                           │
│  ├── gmail/data.db                              │
│  ├── whatsapp/data.db                           │
│  ├── history/data.db                            │
│  ├── user_memory/data.db                        │
│  └── applenotes/data.db                         │
└─────────────────────────────────────────────────┘
```

### Local Machine

```
┌─────────────────────────────────────────────────┐
│  obk CLI (controller, mode: remote)             │
│  ├── All commands proxy to remote server API    │
│  ├── obk db gmail "SQL"  → POST /api/db/gmail   │
│  ├── obk memory list     → GET  /api/memory     │
│  ├── obk whatsapp send   → POST /api/whatsapp   │
│  ├── obk status          → GET  /api/health     │
│  └── obk setup           → configures remote    │
│                                                  │
│  obk daemon --bridge (optional, macOS only)     │
│  └── Apple Notes sync → push to remote API      │
│                                                  │
│  Claude Code (optional, personal subscription)  │
│  └── Same skills, obk commands proxy to remote  │
└─────────────────────────────────────────────────┘
```

### What runs where

| Component | Local mode | Server (Docker) | Remote (local machine) |
|-----------|-----------|-----------------|----------------------|
| WhatsApp sync | local daemon | server | — |
| Gmail sync | local daemon | server | — |
| History DB | local | server | — |
| User memory | local | server | — |
| Apple Notes | local daemon | — | bridge → push to server |
| SQLite databases | ~/.obk/ | /data/ (volume) | — |
| Telegram bot | — | server | — |
| Agent | obk chat | Telegram sessions | obk chat (proxied) |
| Claude Code | terminal | — | terminal (proxied) |

## Quick Start

### 1. Deploy the server

Copy files to your server:

```bash
scp -r infrastructure/docker/ user@server:~/obk/
```

Create a `.env` file on the server:

```
OBK_AUTH_USERNAME=obk
OBK_AUTH_PASSWORD=<choose-a-password>
ANTHROPIC_API_KEY=<your-key>
TELEGRAM_BOT_TOKEN=<from-botfather>
TELEGRAM_OWNER_ID=<your-telegram-user-id>
```

Optional provider keys (for multi-model support):

```
OPENAI_API_KEY=<your-key>
GEMINI_API_KEY=<your-key>
```

Start the server:

```bash
ssh user@server "cd obk && docker compose up -d"
```

Verify it's running:

```bash
curl https://server:8443/api/health
```

### 2. Configure local CLI

Run the interactive setup:

```bash
obk setup
```

Select "Remote" deployment, then enter your server URL and credentials. The setup will test the connection and save the config.

Or manually edit `~/.obk/config.yaml`:

```yaml
mode: "remote"

remote:
  server: "https://my-server.example.com:8443"
  username: "obk"
  password: "your-password"
```

### 3. Authenticate data sources

**WhatsApp:**

```bash
obk whatsapp auth login
```

This opens the remote server's QR page in your browser. Scan with your phone.

**Gmail:**

Gmail OAuth is configured during `obk setup` or via the server's Google OAuth flow.

### 4. Apple Notes bridge (optional, macOS only)

If you want Apple Notes synced to the remote server:

```bash
obk service run daemon --bridge
```

This syncs Apple Notes every 30 seconds and pushes changes to the remote server.

## CLI Command Proxying

When `mode: remote` is set, all `obk` commands automatically proxy to the server:

| Command | Local mode | Remote mode |
|---------|-----------|-------------|
| `obk db gmail "SQL"` | sqlite3 locally | POST /api/db/gmail |
| `obk db whatsapp "SQL"` | sqlite3 locally | POST /api/db/whatsapp |
| `obk db history "SQL"` | sqlite3 locally | POST /api/db/history |
| `obk db user_memory "SQL"` | sqlite3 locally | POST /api/db/user_memory |
| `obk whatsapp send` | sends locally | POST /api/whatsapp/send |
| `obk gmail send` | sends locally | POST /api/gmail/send |
| `obk gmail sync` | syncs locally | POST /api/gmail/sync |
| `obk memory list` | reads locally | GET /api/memory |
| `obk memory add` | writes locally | POST /api/memory |
| `obk memory delete` | deletes locally | DELETE /api/memory/{id} |
| `obk memory extract` | runs locally | POST /api/memory/extract |
| `obk whatsapp auth login` | local QR | opens remote /auth/whatsapp |
| `obk status` | checks local | GET /api/health |

Skills (email-read, whatsapp-read, etc.) use `obk db` instead of raw `sqlite3`, so they work transparently in both modes.

## `obk db` Command

Replaces direct `sqlite3` access in skills. Provides a unified interface regardless of deployment mode.

```bash
obk db gmail "SELECT subject, sender_name, date FROM emails ORDER BY date DESC LIMIT 10"
obk db whatsapp "SELECT text, sender_name FROM whatsapp_messages WHERE chat_jid = '...' LIMIT 20"
obk db history "SELECT session_id, started_at FROM history_conversations ORDER BY updated_at DESC LIMIT 10"
obk db user_memory "SELECT content, category FROM memories ORDER BY updated_at DESC"
obk db applenotes "SELECT title, body FROM notes WHERE title LIKE '%shopping%'"
```

Valid sources: `gmail`, `whatsapp`, `history`, `user_memory`, `applenotes`.

## Telegram Bot

The Telegram bot runs on the server as a long-polling loop (no public URL or webhook needed).

**Key behaviors:**

- **Owner-only**: Messages from non-owner Telegram IDs are ignored
- **Session management**: 15-minute inactivity timeout per session
- **Memory injection**: User memories are loaded from DB and injected into the agent's system prompt before each session
- **History persistence**: All conversations are saved to the history DB
- **Memory extraction**: After a session ends (15 min idle), memories are extracted asynchronously from the conversation
- **Sequential processing**: Messages are processed one at a time to prevent race conditions

**Getting your Telegram owner ID:**

1. Message [@userinfobot](https://t.me/userinfobot) on Telegram
2. It replies with your user ID
3. Set this as `TELEGRAM_OWNER_ID`

**Creating a bot token:**

1. Message [@BotFather](https://t.me/BotFather) on Telegram
2. Send `/newbot` and follow the prompts
3. Copy the token and set it as `TELEGRAM_BOT_TOKEN`

## HTTP API

All endpoints except `/api/health` require basic auth.

Request body limit: 1 MB.

### Health

```
GET /api/health
→ {"status": "ok"}
```

### DB Proxy

Read-only SQL queries against any data source. Rejects non-SELECT queries and stacked queries (semicolons).

```
POST /api/db/{source}
{"sql": "SELECT ..."}
→ {"columns": ["col1", "col2"], "rows": [["val1", "val2"]]}
```

### Memory

```
GET  /api/memory                     → [{"id":1, "content":"...", "category":"...", ...}]
GET  /api/memory?category=preference → filtered list
POST /api/memory                     → {"content":"...", "category":"...", "source":"..."}
                                     ← {"id": 1}
DELETE /api/memory/{id}              → 204 No Content
POST /api/memory/extract             → {"last": 1}
                                     ← {"added":0, "updated":0, "deleted":0, "skipped":0}
```

### Gmail

```
POST /api/gmail/send  → {"to":"...", "subject":"...", "body":"...", "account":"..."}
POST /api/gmail/draft → {"to":"...", "subject":"...", "body":"...", "account":"..."}
POST /api/gmail/sync  → {"full":false, "after":"2024-01-01", "account":"..."}
```

### WhatsApp

```
POST /api/whatsapp/send → {"to":"jid@s.whatsapp.net", "text":"..."}
                        ← {"message_id":"...", "timestamp":"..."}
```

### Apple Notes

```
POST /api/applenotes/push → [{"apple_id":"...", "title":"...", "body":"...", ...}]
                          ← {"saved": 5}
```

### Auth

```
GET /auth/whatsapp         → HTML page with QR code for WhatsApp linking
GET /auth/whatsapp/api/qr  → {"state":"qr", "qr":"..."} or {"state":"authenticated"}
```

## Security

- **Basic auth** on all API endpoints (except health). Uses `crypto/subtle.ConstantTimeCompare` for timing-safe credential comparison.
- **Server refuses to start** without auth credentials configured (via env vars or config).
- **DB proxy is read-only**: Uses `database/sql` with `?mode=ro` (SQLite read-only mode). Rejects non-SELECT queries and stacked queries (semicolons) as defense-in-depth.
- **Request body limit**: 1 MB via `http.MaxBytesReader`.
- **Internal errors are not leaked**: Server logs errors with `slog.Error`, returns generic messages to clients.
- **DB migrations run once at startup**, not per-request.

## Configuration Reference

### Local machine (remote mode)

```yaml
mode: "remote"

remote:
  server: "https://my-server.example.com:8443"
  username: "obk"
  password: "your-password"

# Optional: for local obk chat against remote data
models:
  default: anthropic/claude-sonnet-4-6
  providers:
    anthropic:
      api_key_ref: "keychain:obk/anthropic"
```

### Docker container (server mode)

```yaml
mode: "server"

auth:
  username: "obk"
  password: "..."  # or via OBK_AUTH_PASSWORD env var

channels:
  telegram:
    bot_token: "..."      # or via TELEGRAM_BOT_TOKEN env var
    owner_id: 12345678    # or via TELEGRAM_OWNER_ID env var

models:
  default: anthropic/claude-sonnet-4-6
  complex: anthropic/claude-opus-4-6
  fast: gemini/gemini-2.5-flash
  providers:
    anthropic:
      api_key_ref: ""  # use ANTHROPIC_API_KEY env var
    gemini:
      api_key_ref: ""  # use GEMINI_API_KEY env var

gmail:
  storage:
    driver: sqlite
whatsapp:
  storage:
    driver: sqlite
history:
  storage:
    driver: sqlite
user_memory:
  storage:
    driver: sqlite
applenotes:
  storage:
    driver: sqlite
```

Environment variables take precedence over config file values for auth credentials, API keys, and Telegram settings.

## Docker Compose

```yaml
services:
  obk-server:
    build:
      context: ../..
      dockerfile: infrastructure/docker/Dockerfile
    ports:
      - "8443:8443"
    environment:
      - OBK_CONFIG_DIR=/data
      - OBK_AUTH_USERNAME=${OBK_AUTH_USERNAME}
      - OBK_AUTH_PASSWORD=${OBK_AUTH_PASSWORD}
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
      - OPENAI_API_KEY=${OPENAI_API_KEY:-}
      - GEMINI_API_KEY=${GEMINI_API_KEY:-}
      - TELEGRAM_BOT_TOKEN=${TELEGRAM_BOT_TOKEN}
      - TELEGRAM_OWNER_ID=${TELEGRAM_OWNER_ID}
    volumes:
      - obk-data:/data
    restart: unless-stopped

volumes:
  obk-data:
```

## Apple Notes Bridge

Apple Notes requires macOS (uses AppleScript), so it can't run in the Docker container. The bridge syncs notes from your Mac and pushes them to the remote server.

```bash
obk service run daemon --bridge
```

**Behavior:**
- Requires `mode: remote` in config
- Syncs Apple Notes every 30 seconds
- Tracks last push timestamp to skip unchanged notes
- Only pushes notes modified since the last successful push
- Runs until interrupted (Ctrl+C)

## Operating Modes Summary

```
obk service run daemon              (local mode, default)
├── Full sync: WhatsApp + Gmail + Apple Notes
└── Background tasks

obk service run daemon --bridge     (local machine, remote mode)
├── Apple Notes sync only
└── Pushes to remote API

obk service run server              (inside Docker container)
├── HTTP API (port 8443)
├── Telegram bot
├── Data sync: WhatsApp + Gmail (no Apple Notes)
└── DB migrations at startup
```
