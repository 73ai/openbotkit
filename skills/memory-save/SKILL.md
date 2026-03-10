---
name: memory-save
description: Save and recall personal facts about the user (memories, preferences, relationships)
allowed-tools: Bash(obk *)
---

## When to use
- User explicitly says "remember this", "note that", "don't forget that..."
- User directly introduces themselves ("My name is...", "I work at...")

## When NOT to use
- Don't proactively save facts the user didn't ask you to remember
- Don't save ephemeral or session-specific information
- Don't save information about code or technical solutions

## Commands

### Save a memory
```bash
obk memory add "User prefers dark mode" --category preference
```

### Categories
- identity: name, location, profession, education
- preference: likes, dislikes, communication style
- relationship: people, pets, family
- project: what they're working on, goals, plans

### List existing memories (to avoid duplicates)
```bash
obk memory list
```

### Examples
```bash
obk memory add "User's name is Priyanshu" --category identity
obk memory add "User prefers Go over Python" --category preference
obk memory add "User's dog is named Max" --category relationship
obk memory add "User is building OpenBotKit, a personal assistant" --category project
```
