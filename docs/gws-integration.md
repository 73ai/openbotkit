# Google Workspace Integration

This document records the architectural decisions behind OpenBotKit's integration with Google Workspace via the [gws CLI](https://github.com/googleworkspace/cli).

## Why gws

gws is a Rust-based CLI that generates its command tree from Google Discovery Documents at runtime. It covers 40+ Google Workspace APIs with a unified interface. Rather than building API clients for Calendar, Drive, Docs, Sheets, Tasks, and Contacts individually, we shell out to gws and get all of them for free, including any new APIs Google adds.

The trade-off: we depend on an external binary. We mitigate this by treating gws as a pure executor with no state of its own — obk owns all authentication and authorization.

## Problems with the naive approach

The first version had the agent call `bash("gws calendar events.list ...")` directly. This created five problems that all stem from the same root cause: no control boundary between the agent and gws.

### Dual auth stores

obk and gws each maintained independent Google OAuth tokens. Setup copied OAuth client credentials to gws and ran `gws auth login` as a separate flow. Two consent screens, two token stores, two refresh cycles. Scope drift between them was inevitable.

### Stale skills

Skills were generated once during `obk setup` by introspecting gws's available commands. If gws updated, the user changed scopes, or Google added API methods, the skills became stale. The manifest tracked a gws version but never rechecked it.

### No progressive consent

If a user started with Calendar read-only and later asked the agent to create an event, there was no path to upgrade scopes. The agent either had the scope or it didn't. The user had to re-run full setup.

### No approval gate on writes

The agent called `bash("gws gmail +send ...")` with no approval check. The Channel interface already had `RequestApproval()` with working Telegram and CLI implementations, but nothing in the tool execution path used it.

### No tool-to-user side channel

The Tool interface returns `(string, error)` — the return value goes to the agent. There was no way for a tool to communicate with the user directly. This matters because progressive consent and write approval both require direct user interaction mid-execution, not just before or after.

## Decisions

### Decision 1: obk is the single Google IDP

**Context**: Two auth stores cause scope drift and confuse users with duplicate consent screens.

**Decision**: obk owns all Google tokens in its existing SQLite store (`oauth_tokens` table with `granted_scopes` column). gws never stores tokens. Before every gws command, obk injects a fresh access token via the `GOOGLE_WORKSPACE_CLI_TOKEN` environment variable, which gws checks first in its credential chain.

**Consequences**: Setup no longer runs `gws auth login`. The gws binary is verified on PATH but never asked to authenticate. One token store, one refresh cycle, one source of truth for granted scopes. The `HasScopes` check on the token store becomes the single authority for what the user has consented to.

### Decision 2: Narrow Interactor interface for tool-to-user communication

**Context**: Tools need to talk to users (auth links, approval prompts, status updates) without going through the agent. But passing the full `Channel` interface is dangerous — `Channel` includes `Receive()`, and two goroutines reading from the same channel (SessionManager's main loop and a tool) is a race condition.

**Decision**: Define a narrow `Interactor` interface at the consumer (`agent/tools` package) with three methods: `Notify`, `NotifyLink`, and `RequestApproval`. No `Receive`. An adapter in the `channel` package bridges `Channel` to `Interactor`.

**Alternatives considered**:
- **context.WithValue**: Anti-pattern in Go. Hides the dependency, no compile-time check, any tool can extract it, makes testing harder.
- **Registry middleware**: Can handle pre/post hooks (approval before execution, notification after). But progressive consent is a mid-execution interaction: check scope, send auth link, block, user completes OAuth, resume. This can't be expressed as a pre/post hook.
- **Full Channel access**: Race condition with `Receive()`. Ruled out by the type system — the narrow interface makes the wrong thing impossible rather than just discouraged.

**Consequences**: Tools can talk TO the user but never LISTEN. Only SessionManager reads incoming messages. The Interactor boundary is enforced at compile time.

### Decision 3: All gws commands flow through a single tool

**Context**: If gws commands go through bash, there's no place to insert scope checks, progressive consent, or write approval.

**Decision**: A dedicated `gws_execute` tool is the only path to run gws commands. The bash tool rejects any command starting with `gws ` with an error pointing the agent to `gws_execute`.

The tool's execution flow:
1. Parse the command to identify the service and whether it's a write (the `+` prefix convention from gws)
2. Check if the user has the required scopes
3. If scopes are missing, trigger progressive consent (send auth link, block on ScopeWaiter)
4. If it's a write, request user approval via `GuardedWrite`
5. Inject the token and execute via gws

**Consequences**: Hard boundary — the agent cannot bypass approval by calling bash directly. All gws operations are auditable through a single code path.

### Decision 4: ScopeWaiter for async auth completion

**Context**: When a tool triggers progressive consent, it sends an auth link and blocks. The OAuth callback arrives seconds or minutes later via a different HTTP request and must unblock the tool.

**Decision**: A `ScopeWaiter` struct maps OAuth state parameters to pending channels. The tool calls `Wait(state, timeout, scopes, account)` and blocks. The OAuth callback handler calls `Lookup(state)` to retrieve the scopes and account, exchanges the code, saves the token, then calls `Signal(state, nil)` to unblock the tool.

The OAuth `state` parameter serves double duty: CSRF protection (per RFC 6749 section 10.12) and correlation with the waiting tool. State values are generated with `crypto/rand` to prevent prediction.

**Consequences**: The ScopeWaiter is the only mutable state shared between the HTTP handler and the tool execution path. It's protected by a mutex and communicates via buffered channels. The tool resumes exactly where it left off after consent completes.

### Decision 5: Write detection trusts the `+` prefix only

**Context**: gws uses a `+` prefix convention for write operations (e.g., `+send`, `+insert`). An early version also did keyword matching (`insert`, `create`, `update`, `delete`, `send`, `patch`) which caused false positives — a command like `calendar events.list --query "delete meeting"` would trigger approval.

**Decision**: Write detection checks only for `+` prefixed arguments. No keyword fallback.

**Consequences**: If gws ever introduces a write command without the `+` prefix, it won't get approval-gated. This is an acceptable trade-off: false negatives (missing an approval) are better handled by updating the tool than false positives (blocking reads with approval prompts).

### Decision 6: GuardedWrite as a composable function

**Context**: The approval-then-execute-then-notify pattern will repeat across tools (gws_execute, future email_send, whatsapp_send).

**Decision**: Extract it as a standalone function, not a framework or middleware. `GuardedWrite(ctx, interactor, description, action)` requests approval, executes on approve, notifies the user, and returns `"denied_by_user"` on deny. All Notify errors are propagated, not dropped.

**Consequences**: Tools compose with `GuardedWrite` via a closure. No inheritance, no framework lock-in. Each tool controls exactly when in its execution flow to call it.

### Decision 7: Skill refresh on startup and after OAuth

**Context**: Skills generated during setup become stale when gws updates or scopes change.

**Decision**: Refresh skills at two points: server startup (compare installed gws version with current `gws --version`) and after any OAuth callback (new scopes may unlock previously unavailable skills). The manifest is cached at SessionManager creation to avoid disk reads on every message.

**Consequences**: Skills stay current without user intervention. The startup cost is one `gws --version` call and a manifest comparison.

## Architecture

```
server.go (creates shared resources)
+-- ch = NewChannel(bot, ownerID)
+-- interactor = NewInteractor(ch)          adapts Channel, hides Receive()
+-- scopeWaiter = NewScopeWaiter()          shared with callback route
+-- tokenBridge = NewTokenBridge(google, account)
|
+-- SessionManager
|   +-- newAgent()
|       +-- BashTool                        blocks "gws ..." commands
|       +-- GWSExecuteTool                  scope check -> consent -> approval -> execute
|       |   can: Notify, NotifyLink, RequestApproval
|       |   cannot: Receive (not in Interactor interface)
|       +-- SubagentTool
|
+-- HTTP routes
    +-- /auth/google/callback               ExchangeCode() -> scopeWaiter.Signal()
```

The `scopeWaiter` is the only object shared between the HTTP handler and the tool execution path, protected by a mutex and communicating via buffered channels keyed by the OAuth state parameter.
