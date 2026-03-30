## Commands

### View timeline

```bash
# Show stored posts (requires prior sync)
obk x timeline [--limit 20] [--json]
```

### Sync timeline from X

```bash
# Incremental sync (following timeline)
obk x sync

# Sync "For You" timeline
obk x sync --type foryou

# Full re-sync
obk x sync --full
```

### View a post and its thread

```bash
obk x post show <tweet-id> [--json]
```

Shows the post content, author, stats, and any replies in the thread. Checks local DB first, falls back to the API.

### View replies to a post

```bash
obk x post replies <tweet-id> [--json]
```

Fetches the conversation thread and shows only the replies.

### Search posts

```bash
# Search live on X
obk x search "<query>" [--limit 20] [--json]

# Search local database only
obk x search "<query>" --local [--limit 20] [--json]
```

### View notifications

```bash
obk x notifications [--limit 20] [--json]
```

Fetches recent mentions and notifications from X.

## Examples

```bash
# Check what's new on my timeline
obk x sync && obk x timeline --limit 10

# Find posts about a topic
obk x search "machine learning" --limit 5

# Read a specific post and its replies
obk x post show 1234567890

# See who replied to a post
obk x post replies 1234567890

# Check mentions
obk x notifications --limit 10

# Get post data as JSON for processing
obk x post show 1234567890 --json
```

## Notes

- Requires authenticated X session (`obk x auth login`)
- Timeline data is stored locally after sync for fast access
- Search defaults to live search on X; use `--local` for local-only
