## Commands

### Post a new tweet

```bash
obk x post new "<text>"
```

### Reply to a post

```bash
obk x post reply <tweet-id> "<text>"
```

### Like a post

```bash
obk x post like <tweet-id>
```

### Repost (retweet)

```bash
obk x post repost <tweet-id>
```

## Confirmation rules

- Always confirm the content with the user before posting a new tweet or reply
- For likes and reposts, confirm if the user's intent is ambiguous
- If the user explicitly says "post this" or "tweet this", send immediately
- Never post on behalf of the user without clear intent

## Examples

```bash
# Post a new tweet
obk x post new "Just shipped a new feature!"

# Reply to someone's post
obk x post reply 1234567890 "Great point, thanks for sharing!"

# Like a post
obk x post like 1234567890

# Repost something
obk x post repost 1234567890
```

## Notes

- Requires authenticated X session (`obk x auth login`)
- Posts are public and visible to all X users
- There is no undo for likes/reposts via this CLI (yet)
