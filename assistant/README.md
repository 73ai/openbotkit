# Assistant

Personal assistant powered by Claude Code with access to email, WhatsApp, and conversation history.

## Setup

This directory must be symlinked outside the repo to avoid loading the parent CLAUDE.md.

Claude Code walks up the directory tree and loads all CLAUDE.md files from parent directories.
There is no way to disable this ([open feature request](https://github.com/anthropics/claude-code/issues/20880)).
If you run `claude` directly from `assistant/` inside the repo, it will load the root CLAUDE.md
(dev rules, commit conventions, etc.) into the assistant context, which is not what we want.

```bash
ln -s /path/to/openbotkit/assistant ~/assistant
cd ~/assistant && claude
```

## Prerequisites

- `obk` must be in PATH: `go build -o ~/go/bin/obk .` (from repo root)
- Gmail synced: `obk gmail auth login && obk gmail sync`
- WhatsApp synced: `obk whatsapp auth login && obk whatsapp sync`

## What's inside

- `CLAUDE.md` — personal assistant persona
- `.claude/settings.json` — Stop hook that captures conversations via `obk memory capture`
- `.claude/skills/email-read/` — Gmail query patterns
- `.claude/skills/whatsapp-read/` — WhatsApp query patterns
- `.claude/skills/memory-read/` — conversation history query patterns
