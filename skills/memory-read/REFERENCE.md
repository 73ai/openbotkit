## When to use
- User asks about something they previously told you ("What's my dog's name?", "Who is Raj?")
- User asks you to check memories ("Check my memories for...", "Do you remember...")
- You need context about the user's preferences, relationships, or projects
- ALWAYS run `obk memory list` to check what's stored — the command returns ALL saved facts

## Commands

### List all memories
```bash
obk memory list
```

### List memories by category
```bash
obk memory list --category relationship
obk memory list --category project
```

### Categories
- identity: name, location, profession, education
- preference: likes, dislikes, communication style
- relationship: people, pets, family
- project: what they're working on, goals, plans

### Examples
```bash
# List everything stored
obk memory list

# Filter by category
obk memory list --category relationship
obk memory list --category project
```
