# Changelog

All notable changes to OpenBotKit are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/).

## 2026-03-22

### Added

- **Backup system**: full backup and restore for all SQLite databases, config, and learnings.
- **Cloud backends**: Cloudflare R2, any S3-compatible storage, and Google Drive.
- **Backup wizard**: interactive setup in the settings TUI walks through credentials and verification.
- **Backup daemon**: automatic scheduled backups run in the background.
- **CLI commands**: `obk backup now`, `obk backup list`, `obk backup status`, `obk backup restore`.

## 2026-03-20

### Added

- **Cerebras provider**: new LLM provider with fast inference.
- **Z.AI provider**: GLM models via Z.AI API.
- **Free tier**: Gemini + Cerebras profile with zero API costs to get started.
- **Reactive scheduler**: triggers that fire in response to data changes, not just cron schedules.
- **Budget tracking**: track and limit LLM spending per session.

### Changed

- **History format**: migrated conversation history from SQLite to JSONL files for portability.
- **Memory format**: migrated user memory from SQLite to Markdown files.

## 2026-03-19

### Added

- **Settings TUI**: interactive terminal UI for configuring providers, profiles, sources, and backup.
- **Learnings system**: assistant extracts and remembers facts about you across conversations, stored as local Markdown files, searchable via CLI and tools.
- **Model cache**: providers now cache available models locally for faster startup.

### Fixed

- **Telegram WebView auth**: OAuth consent links now open correctly inside Telegram's in-app browser.

## 2026-03-18

### Added

- **Bash sandboxing**: bash commands run inside a sandbox (Seatbelt on macOS, bwrap on Linux).
- **File approval gates**: `file_write` and `file_edit` require explicit user approval.
- **sandbox_exec tool**: sandboxed code execution for untrusted scripts.

### Changed

- **Source and service refactor**: reorganized source packages and unified service commands into `obk service`.

## 2026-03-17

### Added

- **Website launch**: Astro + Tailwind site with capabilities and mission pages at openbotkit.dev.
- **macOS native integration**: Swift helper binary (`obkmacos`) for Apple Notes, Contacts, and iMessage.
- **Installer script**: `curl -fsSL .../install.sh | sh` for one-command setup.

## 2026-03-15

### Added

- **Google Workspace integration**: 15 services (Calendar, Gmail, Docs, Sheets, Slides, Drive, Meet, Chat, Tasks, Keep, Forms, Classroom, People, Admin Reports, Workflows) with progressive consent.
- **LLM profiles**: starter, standard, and premium tiers with cost guidance.
- **Groq provider**: fast open-source model inference.
- **OpenRouter provider**: access to 100+ models through a single API key.

### Changed

- **Delegation improvements**: file-based delegation results, improved prompt guidance, and full-permission flags for delegate tasks.

## 2026-03-14

### Added

- **Tool output management**: structured tool output handling with size limits and truncation.
- **Nano tier**: ultra-cheap LLM tier for feedback extraction and memory reconciliation.

### Fixed

- **Provider timeouts**: added 60-second HTTP timeouts to OpenAI, Anthropic, and Gemini providers to prevent hanging requests.
