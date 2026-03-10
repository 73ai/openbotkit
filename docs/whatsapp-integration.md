# WhatsApp Integration

## WhatsApp is a data source, not a channel

This is the most important thing to understand about how OpenBotKit uses WhatsApp.

Telegram is a channel — users interact with the assistant through it. WhatsApp is not. WhatsApp is a data source, like Gmail. We sync your messages into a local SQLite database so the assistant can search and read them. That's the primary use case.

Why not use WhatsApp as a bot channel like Telegram? Because WhatsApp explicitly prohibits it. The integrations we use (linked device protocol via whatsmeow) register as a companion device to your personal account. Using that connection to run an automated bot — auto-replying, sending unsolicited messages, running agent loops — violates WhatsApp's Terms of Service and will get your account banned.

We've seen this happen. Projects that treat WhatsApp as a bot platform eventually get their users' accounts blocked. We won't do that.

### How we use WhatsApp

**Read**: The assistant can search and read your WhatsApp messages, just like it searches your email. This is the core use case — "what did David say about the meeting?", "find the address someone sent me last week."

**Send (with approval)**: When the assistant needs to send a WhatsApp message on your behalf, it doesn't send it directly. It drafts the message and presents it to you on Telegram (or CLI) with approve/deny buttons. You review it, you approve it, then it sends. This keeps the interaction human-initiated and within WhatsApp's intended use.

**No automation**: We don't auto-reply to incoming WhatsApp messages. We don't run background loops that send messages. We don't use WhatsApp as an interaction channel for the agent. The agent talks to you on Telegram or CLI. WhatsApp is just data.

## Technical research

Research conducted on 2026-02-28 to evaluate approaches for building the WhatsApp sync engine.

### Libraries evaluated

| Library | Language | Approach | Stars | URL |
|---|---|---|---|---|
| whatsmeow | Go | Direct protocol (WebSocket) | ~4k | https://go.mau.fi/whatsmeow |
| whatsapp-web.js | Node.js | Browser automation (Puppeteer) | ~15k | https://github.com/pedroslopez/whatsapp-web.js |
| Baileys | TypeScript | Direct protocol (WebSocket) | ~8.4k | https://github.com/WhiskeySockets/Baileys |
| wacli | Go (uses whatsmeow) | CLI tool | - | https://github.com/steipete/wacli |

### Three approaches to WhatsApp Web integration

#### 1. Direct protocol implementation (whatsmeow, Baileys)

Both whatsmeow and Baileys implement the WhatsApp Web multi-device protocol directly:

- Open a WebSocket to `wss://web.whatsapp.com/ws/chat`
- Implement the Noise Protocol Framework handshake (`Noise_XX_25519_AESGCM_SHA256`)
- Handle Signal Protocol encryption (Double Ratchet, X25519, AES-256-GCM) for E2E
- Serialize/deserialize WhatsApp's custom binary node format (protobuf-based)
- Register as a linked device (like WhatsApp Web/Desktop)
- Handle media encryption/decryption (AES-256-CBC with HKDF-derived keys)

#### 2. Browser automation (whatsapp-web.js)

whatsapp-web.js takes a fundamentally different approach:

- Launches a real Chromium browser via Puppeteer
- Navigates to `https://web.whatsapp.com/`
- Injects JavaScript to hook into WhatsApp Web's internal webpack modules
- Bridges internal events to Node.js via `page.exposeFunction()` and `page.evaluate()`
- All API calls execute inside the real WhatsApp Web client

### Detailed library analysis

#### whatsmeow (Go) — Chosen

**Repository**: https://go.mau.fi/whatsmeow
**Maintainer**: Tulir Asokan (mautrix ecosystem)
**Used by**: mautrix-whatsapp (Matrix-WhatsApp bridge), wacli

**Authentication**
- QR code pairing: one-time scan, session persisted in SQLite (`sqlstore`)
- Session stores: device identity, Signal protocol keys, signed pre-keys, app state sync keys
- Once paired, reconnects without QR indefinitely (as long as phone comes online every ~14 days)

**Message sync**
- `events.HistorySync`: batch history delivered by WhatsApp on connection (bootstrap sync)
- `events.Message`: individual live messages while connected
- On-demand history backfill via `BuildHistorySyncRequest()` sent as peer message to phone
- Handles: text, extended text, images, video, audio, documents, stickers, reactions, replies

**Media handling**
- Download: `DownloadMediaWithPathToFile()` handles encrypted media download and decryption
- Upload: `Upload()` for encrypting and uploading media to WhatsApp CDN
- Media metadata (direct path, media key, file hashes) stored for deferred download

**Reconnection**
- Library provides connection events; application handles reconnection logic
- wacli demonstrates exponential backoff pattern: 2s -> 4s -> 8s -> ... -> 30s cap
- `sync --follow` mode handles disconnects automatically

**Strengths**
- Native Go — integrates directly into our codebase
- Lightweight — single binary, ~20-50MB memory
- Battle-tested by mautrix-whatsapp (thousands of 24/7 bridge deployments)
- Clean API with typed Go structs for all message types
- SQLite session store fits our existing DB patterns
- No external runtime dependencies

**Weaknesses**
- Must wait for maintainer updates when WhatsApp changes protocol (historically days, not weeks)
- Does not handle polls, ephemeral message nuances, and some newer types
- Phone must be online for history backfill requests

**Reference implementation: wacli**

wacli (https://github.com/steipete/wacli) is a full CLI built on whatsmeow that demonstrates:

- **Two separate SQLite databases**: `session.db` (whatsmeow-owned) and `wacli.db` (app-owned)
- **Three sync modes**: bootstrap (initial), once (sync then exit), follow (continuous daemon)
- **Single-instance file locking** to prevent session conflicts
- **Upsert-everywhere idempotency**: `INSERT ... ON CONFLICT ... DO UPDATE`
- **Media download workers**: 4 concurrent goroutines processing a buffered channel
- **FTS5 full-text search** with graceful fallback to LIKE queries
- **Schema migrations**: version-tracked incremental changes

Architecture:
```
cmd/wacli/       — CLI commands (Cobra)
internal/app/    — Application logic, sync orchestration
internal/wa/     — whatsmeow client wrapper (mutex-protected)
internal/store/  — SQLite persistence layer
internal/lock/   — File locking
internal/config/ — Configuration
```

#### Baileys (TypeScript)

**Repository**: https://github.com/WhiskeySockets/Baileys
**Status**: v7.0.0-rc.9 (still in release candidate)

Same direct WebSocket protocol implementation as whatsmeow, but in TypeScript.

**Additional features over whatsmeow**: Newsletter/channel support, communities support, app state sync with LT-Hash verification, pairing codes (alternative to QR), event buffering for batch processing during history sync.

**Concerns**
- Still in RC status with reported memory leaks (#2344)
- Connection stability issues in multiple open issues
- Requires Rust WASM module (`whatsapp-rust-bridge`) with SIMD CPU support
- Protocol version hardcoded — fails with HTTP 405 when stale
- `useMultiFileAuthState` explicitly marked "not production-ready"
- Would require Node.js sidecar process to integrate with our Go codebase

#### whatsapp-web.js (Node.js)

**Repository**: https://github.com/pedroslopez/whatsapp-web.js

**Strengths**: Runs the real WhatsApp Web client — complete message type coverage. Protocol changes don't break the protocol layer (but internal module hooks still break). Media decryption uses WhatsApp's own code. Rich API for all WhatsApp features.

**Weaknesses**: Resource-heavy (~500MB+ RAM per instance, full Chromium). Fragile internal module hooking (`WAWebCollections`, `WAWebSocketModel`, etc.). Chromium can crash, leak memory, hang over long uptimes. Requires Node.js runtime + Chromium on server. No independent message storage — relies on browser in-memory cache. Would require Node.js sidecar process + IPC to integrate with Go.

### WhatsApp multi-device architecture

**How linked devices work**
- Multi-device (since 2021): linked devices operate independently of the phone
- Each linked device has its own Signal Protocol encryption keys
- Each device maintains its own WebSocket connection to WhatsApp servers
- Up to 4 linked devices per account

**Session lifetime**
- One-time QR code scan for initial pairing
- Session persists indefinitely after that
- No token rotation or refresh flow

**What kills a session**
1. Phone offline for ~14 days (server-enforced expiry)
2. User manually unlinks device from phone settings
3. Session conflict (same device identity from two processes)
4. TOS block / account ban
5. Explicit logout call

**Phone requirements**
- NOT needed for normal operation (send/receive messages)
- Needed for: initial QR pairing, periodic keepalive (~every 14 days), history backfill
- Phone just needs to connect to internet briefly; doesn't need to stay online

## Decision: whatsmeow

**Over Baileys**: Native Go (no sidecar), more stable (Baileys still RC with memory leaks), simpler deployment (no Rust WASM dependency), battle-tested by mautrix-whatsapp in production bridge scenarios.

**Over whatsapp-web.js**: No Chromium overhead (~20MB vs ~500MB RAM), no browser process to crash/hang during long uptimes, simpler deployment and monitoring (single binary), native Go integration. The "protocol resilience" advantage is overstated — whatsapp-web.js also breaks when WhatsApp renames internal webpack modules.

**For our use case**: We need a reliable sync engine that stays connected and fetches messages into SQLite. whatsmeow does exactly this — raw WebSocket with exponential backoff for connectivity, `events.HistorySync` + `events.Message` for message fetching, `client.SendMessage()` for approved sends. Single Go binary, minimal resources, no external runtimes.
