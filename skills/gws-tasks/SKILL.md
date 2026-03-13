---
name: gws-tasks
version: 1.0.0
description: "Google Tasks: Manage task lists and tasks."
metadata:
  openclaw:
    category: "productivity"
    requires:
      bins: ["gws"]
    cliHelp: "gws tasks --help"
---

# tasks (v1)

> **PREREQUISITE:** Read `../gws-shared/SKILL.md` for auth, global flags, and security rules. If missing, run `gws generate-skills` to create it.

```bash
gws tasks <resource> <method> [flags]
```

## API Resources

### tasklists

  - `delete` — Deletes the authenticated user's specified task list. If the list contains assigned tasks, both the assigned tasks and the original tasks in the assignment surface (Docs, Chat Spaces) are deleted.
    Required: `tasklist` (string)
    Example: `gws tasks tasklists delete --params '{"tasklist":"..."}'`
  - `get` — Returns the authenticated user's specified task list.
    Required: `tasklist` (string)
    Example: `gws tasks tasklists get --params '{"tasklist":"..."}'`
  - `insert` — Creates a new task list and adds it to the authenticated user's task lists. A user can have up to 2000 lists at a time.
  - `list` — Returns all the authenticated user's task lists. A user can have up to 2000 lists at a time.
    Key params: `maxResults` (int32), `pageToken` (string)
    Example: `gws tasks tasklists list --params '{"maxResults":"...","pageToken":"..."}'`
  - `patch` — Updates the authenticated user's specified task list. This method supports patch semantics.
    Required: `tasklist` (string)
    Example: `gws tasks tasklists patch --params '{"tasklist":"..."}'`
  - `update` — Updates the authenticated user's specified task list.
    Required: `tasklist` (string)
    Example: `gws tasks tasklists update --params '{"tasklist":"..."}'`

### tasks

  - `clear` — Clears all completed tasks from the specified task list. The affected tasks will be marked as 'hidden' and no longer be returned by default when retrieving all tasks for a task list.
    Required: `tasklist` (string)
    Example: `gws tasks tasks clear --params '{"tasklist":"..."}'`
  - `delete` — Deletes the specified task from the task list. If the task is assigned, both the assigned task and the original task (in Docs, Chat Spaces) are deleted. To delete the assigned task only, navigate to the assignment surface and unassign the task from there.
    Required: `task` (string), `tasklist` (string)
    Example: `gws tasks tasks delete --params '{"task":"...","tasklist":"..."}'`
  - `get` — Returns the specified task.
    Required: `task` (string), `tasklist` (string)
    Example: `gws tasks tasks get --params '{"task":"...","tasklist":"..."}'`
  - `insert` — Creates a new task on the specified task list. Tasks assigned from Docs or Chat Spaces cannot be inserted from Tasks Public API; they can only be created by assigning them from Docs or Chat Spaces. A user can have up to 20,000 non-hidden tasks per list and up to 100,000 tasks in total at a time.
    Required: `tasklist` (string)
    Key params: `parent` (string), `previous` (string)
    Example: `gws tasks tasks insert --params '{"tasklist":"..."}'`
  - `list` — Returns all tasks in the specified task list. Doesn't return assigned tasks by default (from Docs, Chat Spaces). A user can have up to 20,000 non-hidden tasks per list and up to 100,000 tasks in total at a time.
    Required: `tasklist` (string)
    Key params: `completedMax` (string), `completedMin` (string), `dueMax` (string), `dueMin` (string), `maxResults` (int32)
    Example: `gws tasks tasks list --params '{"tasklist":"..."}'`
  - `move` — Moves the specified task to another position in the destination task list. If the destination list is not specified, the task is moved within its current list. This can include putting it as a child task under a new parent and/or move it to a different position among its sibling tasks. A user can have up to 2,000 subtasks per task.
    Required: `task` (string), `tasklist` (string)
    Key params: `destinationTasklist` (string), `parent` (string), `previous` (string)
    Example: `gws tasks tasks move --params '{"task":"...","tasklist":"..."}'`
  - `patch` — Updates the specified task. This method supports patch semantics.
    Required: `task` (string), `tasklist` (string)
    Example: `gws tasks tasks patch --params '{"task":"...","tasklist":"..."}'`
  - `update` — Updates the specified task.
    Required: `task` (string), `tasklist` (string)
    Example: `gws tasks tasks update --params '{"task":"...","tasklist":"..."}'`

## Discovering Commands

Before calling any API method, inspect it:

```bash
# Browse resources and methods
gws tasks --help

# Inspect a method's required params, types, and defaults
gws schema tasks.<resource>.<method>
```

Use `gws schema` output to build your `--params` and `--json` flags.

