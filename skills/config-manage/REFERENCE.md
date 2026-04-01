## When to use
- User asks to change settings, configuration, timezone, workspace, models, providers, etc.
- User asks to view current configuration

## Commands

### View all config
```bash
obk config list
```

### Set a config value
```bash
obk config set <key> <value>
```

Keys use dotted paths matching the YAML structure. Examples:

```bash
# Top-level keys
obk config set timezone America/New_York
obk config set workspace /path/to/workspace
obk config set mode local

# Nested keys (follow the YAML hierarchy)
obk config set models.default anthropic/claude-sonnet-4-6
obk config set models.fast gemini/gemini-2.5-flash
obk config set gmail.storage.driver postgres
obk config set gmail.storage.dsn postgres://localhost/gmail
obk config set gmail.download_attachments true
obk config set gmail.sync_days 30
obk config set daemon.gmail_sync_period 30m
obk config set providers.google.credentials_file /path/to/creds.json
obk config set integrations.gws.callback_url https://example.com/callback
obk config set integrations.gws.ngrok_domain example.ngrok-free.app
```

### Show config directory path
```bash
obk config path
```

## How it works
- `obk config set` accepts any dotted path matching the config YAML structure
- Nested pointer fields (e.g., `providers.google`) are automatically allocated if nil
- Supported value types: string, bool (`true`/`false`), int, float
- Timezone values are validated via `time.LoadLocation`

## Tips
- Run `obk config list` first to see the current YAML structure and discover available keys
- Keys match the YAML tag names, not the Go struct field names
