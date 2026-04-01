## Skill Format Spec

Every skill has two files in its directory:

### SKILL.md
YAML frontmatter + short summary. Keep it slim — under 10 lines.

```yaml
---
name: my-skill
description: One-line description (used for search/index)
allowed-tools: Bash(obk *), file_read
---

Short summary of what this skill does.

Read the REFERENCE.md in this skill's directory for full instructions.
```

### REFERENCE.md
Full instructions, commands, examples, and notes. This is loaded on demand when the skill is activated.

Structure:
- `## When to use` — triggers for this skill
- `## When NOT to use` — avoid false positives
- `## Commands` — CLI commands with bash examples
- `## Examples` — concrete usage patterns
- `## Notes` — edge cases, limitations

### Confirmation rules for write operations
If the skill performs write/destructive operations, add a `[!CAUTION]` marker:

```markdown
> [!CAUTION]
> This skill modifies/deletes data. Always confirm with the user before executing write commands.
```

## CLI Commands

### List all skills
```bash
obk skills list
```

### Show a skill's content
```bash
obk skills show <name>
```

### Create a custom skill
```bash
obk skills create <name> --skill-file /path/to/SKILL.md --ref-file /path/to/REFERENCE.md
```

### Update an existing custom/external skill
```bash
obk skills update <name> --skill-file /path/to/SKILL.md --ref-file /path/to/REFERENCE.md
```

### Remove a custom/external skill
```bash
obk skills remove <name>
```

## Workflow: Create a Skill from Scratch

1. Research what the skill needs (web search, explore docs, read existing skills for patterns)
2. Write SKILL.md to the workspace staging area:
   ```bash
   file_write <workspace>/staging/SKILL.md
   ```
3. Write REFERENCE.md to the workspace staging area:
   ```bash
   file_write <workspace>/staging/REFERENCE.md
   ```
4. Install:
   ```bash
   obk skills create <name> --skill-file <workspace>/staging/SKILL.md --ref-file <workspace>/staging/REFERENCE.md
   ```
5. Verify:
   ```bash
   obk skills show <name>
   ```
   Also verify with `search_skills("<name>")` and `load_skills(["<name>"])`.
6. Clean up staging files.

## Workflow: Install from External Repo

1. Clone to staging:
   ```bash
   git clone --depth 1 <url> <workspace>/staging/<repo-name>
   ```
2. Explore: read README, list files, understand the repo's structure and capabilities
3. For each capability found: write SKILL.md + REFERENCE.md adapting the repo's docs into obk skill format
4. Install each skill:
   ```bash
   obk skills create <name> --skill-file <workspace>/staging/SKILL.md --ref-file <workspace>/staging/REFERENCE.md
   ```
5. Clean up:
   ```bash
   rm -rf <workspace>/staging/<repo-name>
   ```

## Workflow: Build a Tool + Skill

1. Identify what the tool needs to do
2. Create tool directory: `<workspace>/tools/<tool-name>/`
3. Implement the tool (bash script, Python script, Go binary, etc.)
4. Build/install as needed: `go build`, `pip install`, `chmod +x`, etc.
5. Create a skill that teaches the agent how to invoke the new tool via bash
6. Verify: run the tool manually, load the skill, test end-to-end

## Workspace Layout

```
<workspace>/
  staging/           # Temporary work area for skill files and repo clones
  tools/             # Built tools that skills reference
    <tool-name>/
      main.py        # or main.go, script.sh, etc.
```

## Best Practices

- Keep SKILL.md frontmatter minimal — name, description, allowed-tools
- Put all detail in REFERENCE.md — commands, examples, schema, notes
- Include concrete CLI examples in REFERENCE.md (copy-paste ready)
- Skills should be self-contained: all info needed to use them in REFERENCE.md
- Use `[!CAUTION]` for any skill that writes, deletes, or modifies data
- Add confirmation rules for ambiguous actions (e.g., "if the user says 'clean up' — confirm what to delete")
- Reference existing skills (`obk skills show <name>`) as templates

## Complete Examples

### Example 1: Read skill (query SQLite database)

**SKILL.md:**
```yaml
---
name: sqlite-query
description: Query local SQLite databases using sqlite3 CLI
allowed-tools: Bash(sqlite3 *), Bash(ls *)
---

Query SQLite databases using the sqlite3 command-line tool.

Read the REFERENCE.md in this skill's directory for full instructions.
```

**REFERENCE.md:**
```markdown
## When to use
- User asks to query, search, or explore a .db file
- User asks about data in a SQLite database

## Commands

### List tables
` + "```" + `bash
sqlite3 /path/to/db.db ".tables"
` + "```" + `

### Query data
` + "```" + `bash
sqlite3 -header -column /path/to/db.db "SELECT * FROM table LIMIT 20;"
` + "```" + `

### Show schema
` + "```" + `bash
sqlite3 /path/to/db.db ".schema table_name"
` + "```" + `
```

### Example 2: Write skill with [!CAUTION]

**SKILL.md:**
```yaml
---
name: file-cleanup
description: Find and delete temporary or old files from directories
allowed-tools: Bash(find *), Bash(rm *)
---

Clean up temporary and old files from directories.

Read the REFERENCE.md in this skill's directory for full instructions.
```

**REFERENCE.md:**
```markdown
> [!CAUTION]
> This skill deletes files permanently. Always confirm with the user before running rm commands.
> List files first with find, show the list, and get explicit approval before deleting.

## Commands

### Find old files (dry run)
` + "```" + `bash
find /path -name "*.tmp" -mtime +30
` + "```" + `

### Delete after confirmation
` + "```" + `bash
find /path -name "*.tmp" -mtime +30 -delete
` + "```" + `
```

### Example 3: Skill wrapping an external CLI tool

**SKILL.md:**
```yaml
---
name: csv-to-json
description: Convert CSV files to JSON using a workspace tool
allowed-tools: Bash(python3 *), file_read
---

Convert CSV files to JSON format using a Python script in the workspace.

Read the REFERENCE.md in this skill's directory for full instructions.
```

**REFERENCE.md:**
```markdown
## When to use
- User asks to convert CSV to JSON
- User has a .csv file and wants structured data

## Commands

### Convert a CSV file
` + "```" + `bash
python3 <workspace>/tools/csv-to-json/convert.py /path/to/input.csv /path/to/output.json
` + "```" + `

### Convert with custom delimiter
` + "```" + `bash
python3 <workspace>/tools/csv-to-json/convert.py --delimiter ";" /path/to/input.csv /path/to/output.json
` + "```" + `
```
