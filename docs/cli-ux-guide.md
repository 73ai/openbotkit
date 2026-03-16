# CLI UX Guide

Conventions for the `obk` CLI. Follow these when adding or modifying commands.

## Command Hierarchy

Resource-first structure, modeled after gcloud/aws/kubectl:

```
obk <resource> <action> [args] [flags]
```

Provider commands (auth + data) live under one top-level group:

```
obk gmail auth login
obk gmail emails list
obk whatsapp messages list
obk slack channels
```

Core commands live at the root:

```
obk chat
obk status
obk doctor
obk setup
```

## Standard Verbs

| Verb | Use for | Example |
|------|---------|---------|
| `list` | Collections, multiple items | `obk gmail emails list` |
| `describe` | Single item detail view | `obk config profiles describe claude-all` |
| `get` | Single item by ID | `obk gmail emails get <id>` |
| `create` | Create a new resource | `obk config profiles create` |
| `delete` | Remove a resource | `obk memory delete 42` |
| `search` | Full-text search | `obk gmail emails search "quarterly report"` |
| `sync` | Data ingestion | `obk gmail sync` |
| `login` | Authenticate | `obk gmail auth login` |
| `logout` | Remove credentials | `obk slack auth logout` |
| `send` | Send a message | `obk gmail send --to user@example.com` |

## Required Flags

### `--json` (bool)

All commands that return data must support `--json` for structured output.

```go
cmd.Flags().Bool("json", false, "Output as JSON")
```

When `--json` is set, output only valid JSON to stdout. No decorative text.

### `--limit` (int, default 50)

Commands that return lists should support `--limit`.

```go
cmd.Flags().Int("limit", 50, "Maximum number of results")
```

### `--force` (bool)

Destructive commands must prompt for confirmation by default and support `--force` to skip:

```go
force, _ := cmd.Flags().GetBool("force")
if !force {
    fmt.Printf("About to delete %s. Continue? (y/N): ", target)
    var confirm string
    fmt.Scanln(&confirm)
    if confirm != "y" && confirm != "Y" {
        fmt.Println("Cancelled.")
        return nil
    }
}
```

### `--account` (string)

Commands that operate on provider accounts should accept `--account` to filter by email/identifier.

## Destructive Commands

Commands that delete data, remove credentials, or uninstall services are destructive. They must:

1. Prompt for confirmation by default
2. Accept `--force` to skip the prompt
3. Print what will be affected before asking

Destructive commands include: `delete`, `logout`, `revoke`, `uninstall`.

## Help Examples

Every leaf command (any command with a `RunE`) must have a cobra `Example:` field.

```go
var myCmd = &cobra.Command{
    Use:     "list",
    Short:   "List items",
    Example: `  obk resource list --limit 10
  obk resource list --json`,
    RunE: func(cmd *cobra.Command, args []string) error { ... },
}
```

Rules:
- Indent examples with 2 spaces
- Show 2-3 realistic flag combinations
- Use only flags that exist on the command

## Short Descriptions

Follow the `"Verb noun phrase"` pattern, start with a capital letter:

```
Good:  "List authenticated Google accounts and scopes"
Good:  "Manage Slack data source"
Bad:   "slack workspace commands"
Bad:   "shows the config"
```

## Flag Conventions Summary

| Flag | Type | Default | Purpose |
|------|------|---------|---------|
| `--json` | bool | false | Structured JSON output |
| `--limit` | int | 50 | Max results for list commands |
| `--force` | bool | false | Skip confirmation on destructive ops |
| `--account` | string | — | Filter by provider account |
| `--follow`, `-f` | bool | false | Stream/tail output |
