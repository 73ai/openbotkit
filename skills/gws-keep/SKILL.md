---
name: gws-keep
version: 1.0.0
description: "Manage Google Keep notes."
metadata:
  openclaw:
    category: "productivity"
    requires:
      bins: ["gws"]
    cliHelp: "gws keep --help"
---

# keep (v1)

> **PREREQUISITE:** Read `../gws-shared/SKILL.md` for auth, global flags, and security rules. If missing, run `gws generate-skills` to create it.

```bash
gws keep <resource> <method> [flags]
```

## API Resources

### media

  - `download` — Gets an attachment. To download attachment media via REST requires the alt=media query parameter. Returns a 400 bad request error if attachment media is not available in the requested MIME type.
    Required: `name` (string)
    Key params: `mimeType` (string)
    Example: `gws keep media download --params '{"name":"..."}'`

### notes

  - `create` — Creates a new note.
  - `delete` — Deletes a note. Caller must have the `OWNER` role on the note to delete. Deleting a note removes the resource immediately and cannot be undone. Any collaborators will lose access to the note.
    Required: `name` (string)
    Example: `gws keep notes delete --params '{"name":"..."}'`
  - `get` — Gets a note.
    Required: `name` (string)
    Example: `gws keep notes get --params '{"name":"..."}'`
  - `list` — Lists notes. Every list call returns a page of results with `page_size` as the upper bound of returned items. A `page_size` of zero allows the server to choose the upper bound. The ListNotesResponse contains at most `page_size` entries. If there are more things left to list, it provides a `next_page_token` value. (Page tokens are opaque values.) To get the next page of results, copy the result's `next_page_token` into the next request's `page_token`.
    Key params: `filter` (string), `pageSize` (int32), `pageToken` (string)
    Example: `gws keep notes list --params '{"filter":"...","pageSize":"..."}'`
  - `permissions` — Operations on the 'permissions' resource

## Discovering Commands

Before calling any API method, inspect it:

```bash
# Browse resources and methods
gws keep --help

# Inspect a method's required params, types, and defaults
gws schema keep.<resource>.<method>
```

Use `gws schema` output to build your `--params` and `--json` flags.

