# memory

Personal memory system for OpenBotKit. Stores durable facts about the user — preferences, identity, relationships, projects — that persist across conversations and personalize the assistant.

This is distinct from `source/history/`, which stores conversation transcripts.

For detailed design rationale, research, and references see [docs/memory-system-design.md](../docs/memory-system-design.md).

## Data Model

Each memory is an atomic fact with a category:

| Category       | Examples                                    |
|----------------|---------------------------------------------|
| `identity`     | Name, location, profession, education       |
| `preference`   | Likes dark mode, prefers Go, vegetarian     |
| `relationship` | Wife is Sarah, dog named Max                |
| `project`      | Building OpenBotKit, learning Rust          |

## Store (CRUD)

```go
id, err := memory.Add(db, "User prefers dark mode", memory.CategoryPreference, "manual", "")
err := memory.Update(db, id, "User prefers light mode")
err := memory.Delete(db, id)

m, err := memory.Get(db, id)
all, err := memory.List(db)
prefs, err := memory.ListByCategory(db, memory.CategoryPreference)
results, err := memory.Search(db, "dark mode")
count, err := memory.Count(db)
```

## Extraction Pipeline

Extracts personal facts from conversation messages using an LLM.

```
Messages → preFilter (skip acks, short messages) → LLM extraction → []CandidateFact
```

```go
facts, err := memory.Extract(ctx, llm, messages)
```

## Reconciliation

Reconciles extracted facts against existing memories to avoid duplicates:

1. Search existing memories by keyword
2. No matches → ADD directly
3. Matches found → LLM decides: ADD, UPDATE, DELETE, or NOOP

```go
result, err := memory.Reconcile(ctx, db, llm, candidates)
// result.Added, result.Updated, result.Deleted, result.Skipped
```

## Prompt Injection

Formats memories as markdown for the agent's system prompt:

```go
text := memory.FormatForPrompt(memories)
```

## LLM Interface

Extract and Reconcile use an `LLM` interface for testability:

```go
type LLM interface {
    Chat(ctx context.Context, req provider.ChatRequest) (*provider.ChatResponse, error)
}
```

Production: `&memory.RouterLLM{Router: router, Tier: provider.TierFast}`

Tests: `&mockLLM{response: `[{"content": "...", "category": "..."}]`}`

## CLI Commands

```
obk memory list [--category <cat>]
obk memory add "fact" --category preference
obk memory delete <id>
obk memory extract [--last <n>]
```

## End-to-End Flow

```
Conversation ends
  → hook: obk history capture && obk memory extract &
    → capture: parse transcript → history DB
    → extract (background): messages → LLM extraction → reconcile → memory DB
```
