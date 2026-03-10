# Testing Strategy

## Tests are how we understand the project

Tests are the most essential part of this application. They are not an afterthought or a chore — they are how we document what the system does, how its parts fit together, and how the user experiences it. If you want to understand how openbotkit works, read the tests. If the tests don't make that clear, the tests are wrong.

Every test must have a **point of view**. It must be clear who is doing the observing and what question is being answered. A test without a perspective is a garbage test — it might pass, it might fail, but it doesn't tell you anything.

We avoid vague labels like "e2e" because they don't mean anything on their own. End-to-end of what? From whose perspective? Instead, every test level in this project is defined by its perspective and the question it answers.

## The five perspectives

### 1. Unit tests — the author's perspective

**Question:** Does this piece of logic do what I, the author, intended?

Unit tests are written from the perspective of the developer who wrote the code. You're testing a function, a method, a small unit of logic. You control all inputs, you mock everything external, and you verify that the logic is correct in isolation.

The point of reference is **the code itself**. If `extractContent` receives a protobuf message with an image caption, it should return that caption and `"image"` as the media type. If the retry wrapper gets a 429, it should retry. If it gets a 401, it should not. These tests don't know or care about the rest of the system.

This is where we have the most tests. They're fast, deterministic, and cheap. They run in-memory with mocks, stubs, and `t.TempDir()`. No network, no real databases, no LLM calls.

**What makes a good unit test here:**
- It tests one decision the code makes, not a chain of decisions across modules.
- The test name describes the behavior, not the implementation: `TestRetry_429Retries` not `TestRetryFunction`.
- Mocks are defined in the same test file, not in shared packages. `mockProvider`, `mockBot`, `stubTool` — they live next to what they test.

**Examples:**
- `agent/agent_test.go` — the agent loop with a scripted mock provider. Does it handle tool calls? Does it stop at max iterations? Does it scrub secrets from tool output?
- `memory/store_test.go` — CRUD operations on an in-memory SQLite database. Does dedup work? Does search return the right results?
- `channel/telegram/telegram_test.go` — does Send format Markdown correctly? Does the owner filter reject non-owners?
- `provider/resilient_test.go` — does the retry wrapper honor 429 vs 401 vs max retries?

### 2. Contract tests — the consumer's perspective

**Question:** Does this service honor the promises it makes to its consumers?

Contract tests are written from the perspective of someone who depends on a boundary — an HTTP API, an interface, a protocol. They don't test internal logic. They test that when you call the API in a valid way, you get the right response, and when you call it in an invalid way, you get the right error.

The point of reference is **the contract between two systems**. The server promises: "POST /api/memory with a valid body and auth token creates a memory and returns 200." The contract test verifies that promise holds regardless of what's behind the API.

**Server API contract tests** (`internal/servertest/suite.go`):

The suite is written from the perspective of a client consuming the server's HTTP API. It tests:
- Health checks work without authentication.
- Memory CRUD works with authentication and rejects without it.
- The database proxy rejects non-SELECT queries.
- Send endpoints validate their inputs.

The suite doesn't know the server's internals. It talks to a `Backend` struct with an HTTP client and a `SeedDB` callback for injecting test state. `internal/server/api_test.go` wires this to a real httptest server.

```go
backend := servertest.Backend{
    Client:       authenticatedClient,
    NoAuthClient: unauthenticatedClient,
    SeedDB:       func(source, sql string) error { ... },
}
servertest.Run(t, backend)
```

### 3. Conformance tests — the protocol's perspective

**Question:** Does this implementation conform to the behavioral contract that all implementations must satisfy?

Conformance tests are written from the perspective of the protocol or interface that multiple implementations must honor. We have one agent protocol — tool calls, tool results, streaming — and four LLM providers (Anthropic, Anthropic Vertex, OpenAI, Gemini). Each provider has its own quirks: Gemini needs function names on tool results, OpenAI streams differently, Anthropic Vertex requires project IDs. But from the agent's perspective, they all must behave the same way.

The point of reference is **the shared contract**. The same test runs against every provider:

- `TestProvider_AgentToolExecution` — the provider drives the agent loop: tool call → execution → final response.
- `TestProvider_ToolUseRoundtrip` — tool_use → tool_result → text_response cycle.
- `TestProvider_Streaming` — text deltas arrive followed by a done event.

`EachProvider(t, ...)` runs the test for every provider that has credentials set. Missing credentials skip, not fail. This is how we catch regressions in one provider without touching the others.

### 4. Integration tests — the boundary's perspective

**Question:** When these two bounded contexts work together, does the combined behavior make sense?

Integration tests are written from the perspective of the **boundary between two systems**. Not "does the agent work" and not "does the database work" — but "when a Telegram message arrives, does it flow through the agent and come back as a bot reply with history saved?"

The point of reference is **the seam where two contexts meet**. A Telegram session connects the channel (messages in, replies out) with the agent (LLM reasoning, tool execution) and storage (conversation history, memories). The integration test verifies these pieces compose correctly.

**Session integration tests** (`channel/telegram/integration_test.go`, `source/whatsapp/integration_test.go`):

- `TestSession_MessageAndHistorySaved` — a message enters through the Telegram channel, the agent (with a real LLM) produces a response, the mock bot captures the reply, and the history database has the conversation.
- `TestSession_MemoryInjectedIntoPrompt` — memories from the database appear in the system prompt. This tests that the session correctly bridges the memory store and the LLM provider.
- `TestSession_ToolUseViaBash` — the agent executes bash commands during a session. This tests that tool execution is wired through correctly from channel to agent to tools and back.

Each test creates an isolated environment: temp config directory, migrated databases, mock bot, real LLM provider. The mock bot is the observation point — we verify what the user would actually see.

These use real LLM APIs and skip if credentials are absent.

### 5. Spec tests — the user's perspective

**Question:** Can the agent actually do what the user needs it to do?

Spec tests are written from the perspective of the **user**. Not the developer, not the API consumer, not the protocol — the person who types "find emails from Alice" and expects a useful answer. This is the most important perspective and the hardest to test.

The point of reference is **the user's task**. The user doesn't know about tool registries, skill loading, SQL queries, or bash execution. They know: "I asked for emails from Alice and got them" or "I asked for a summary of all communications and got something useful."

**Why this level exists:** Unit tests can't catch "the skill prompt teaches the LLM wrong SQL." Integration tests can't catch "the agent finds emails but forgets to also check WhatsApp when asked about all communications." Only a test written from the user's perspective can catch these, because only that perspective sees the full chain from question to answer.

**Core principles:**

**Test user behavior, not implementation.** The spec is "find emails from Alice" — not "call search_skills, then load_skills, then bash with a SQL query." If the agent finds a better route to the answer tomorrow, the test should still pass.

**Declarative state setup.** `fixture.GivenEmails(t, emails)` reads like a story. The fixture handles the machinery (SQLite inserts, skill installation, CLI binary builds). The test reads like a user scenario.

**Same spec, multiple providers.** `EachProvider(t, func(t *testing.T, fx *LocalFixture) { ... })` runs the same user scenario against Gemini, Anthropic Vertex, and OpenAI. If the agent can find Alice's emails with Gemini but not OpenAI, that's a real bug.

**Current specs:**

| Spec | The user's task |
|------|----------------|
| `TestSpec_FindEmailsBySender` | "Find emails from Alice" — agent discovers the right skill, queries the database, returns matching emails |
| `TestSpec_SummarizeCommunicationsAcrossSources` | "Summarize all communications with Bob" — agent checks both email and WhatsApp, synthesizes across sources |
| `TestSpec_RecallMemoryAndCorrelateEmails` | "What did I say I cared about, and are there relevant emails?" — agent combines personal memories with email search |

**Assertion strategy for non-deterministic output:**

The LLM is non-deterministic but the data is deterministic. If the agent found the right email, `alice@example.com` appears verbatim in the response. Three assertion layers:

1. **Structural** — what did the agent *do*? Check tool call recordings and send recorders. Free and deterministic.
2. **Substring on data** — does the response contain the database values the agent should have found? Free and deterministic.
3. **LLM judge** — does the response *mean* the right thing? A cheap model evaluates semantic criteria. Used only when substring matching genuinely can't express the assertion.

## Practical guidelines

### When to write which test

| You just... | Write a... | Because... |
|------------|-----------|-----------|
| Wrote a new function | Unit test | You're the author, verify your logic works |
| Added an API endpoint | Contract test | Consumers need to trust the endpoint's promises |
| Added a new LLM provider | Conformance test | It must honor the same protocol as every other provider |
| Wired two subsystems together | Integration test | The boundary between them needs verification |
| Added a new user-facing capability | Spec test | The user's perspective is the only one that can verify it |
| Fixed a bug | Test at the lowest level that reproduces it | Don't use a spec test for what a unit test can catch |

### Naming tests

Test names describe behavior from the test's perspective:
- Unit: `TestRetry_429Retries`, `TestExtractContent_ImageMessage`
- Contract: `TestAPI_MemoryRejectsWithoutAuth`
- Conformance: `TestProvider_ToolUseRoundtrip`
- Integration: `TestSession_MemoryInjectedIntoPrompt`
- Spec: `TestSpec_FindEmailsBySender`

### Graceful skipping

Tests that need external resources (LLM API keys, sqlite3 binary, Docker) skip with `t.Skip()`. This means `go test ./...` always passes on a fresh checkout — unit tests run, everything else skips gracefully. Helpers like `testutil.RequireGeminiKey(t)` standardize this.

### Conventions

- Standard `testing` package only. No test frameworks.
- Mocks live in the test file that uses them.
- Database tests use in-memory SQLite or `t.TempDir()` — no shared state.
- All test assertions are explicit `if` checks with `t.Fatalf`. The failure message says what was expected and what was got.
