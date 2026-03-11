# WebSearch — Strategy & Design Decisions

A Go metasearch data source for OpenBotKit. No API keys. No external services. Just HTTP requests + HTML parsing running locally.

Inspired by [ddgs](https://github.com/deedy5/ddgs) (Python), rebuilt in Go as a first-class openbotkit source.

---

## Problem

Every open-source coding agent delegates web search to their model provider's API. There is no standalone, self-contained, local web search capability that agents can use. We need a Go equivalent integrated into openbotkit — search, news, and page fetching accessible via CLI, consumed by agents as skills.

## Design Principles

1. **Follows openbotkit patterns** — Source interface, CLI commands via `obk websearch`, skills as SKILL.md/REFERENCE.md, SQLite for caching/history.
2. **Zero dependencies on external services** — No API keys, no Docker, no servers. Just HTTP scraping.
3. **Composable** — Search, news, and fetch are separate skills. The agent orchestrates.
4. **Resilient** — Multiple backends with concurrent dispatch, automatic fallback, per-host rate limiting, and health tracking.

---

## Architecture

```
Skills Layer
  web-search, web-fetch, web-news (SKILL.md + REFERENCE.md each)
        │
CLI Layer (obk websearch ...)
  search, news, fetch, backends, history, cache clear
        │
source/websearch/ (Go package)
  ┌─────────────────────────────────────────────┐
  │  Orchestrator                                │
  │  - Backend selection (auto / explicit)       │
  │  - Concurrent dispatch (errgroup)            │
  │  - Result ranking & deduplication            │
  │  - Health tracking (exponential cooldown)    │
  ├─────────────────────────────────────────────┤
  │  Engines (each: HTTP request + HTML parse)   │
  │  DDG, Brave, Mojeek, Wikipedia, Yahoo,       │
  │  Yandex, Google, Bing                        │
  ├─────────────────────────────────────────────┤
  │  httpclient/                                 │
  │  - Wraps internal/browser (utls transport)   │
  │  - UA rotation (4 browser profiles)          │
  │  - Per-host token bucket rate limiting       │
  ├─────────────────────────────────────────────┤
  │  SQLite (cache + history)                    │
  │  search_cache, fetch_cache, search_history   │
  └─────────────────────────────────────────────┘
```

---

## Key Decisions

### 1. Flat package over engine/ subdirectory

**Decision:** Engines live directly in `source/websearch/` (e.g., `duckduckgo.go`, `brave.go`), not in a `source/websearch/engine/` subdirectory.

**Why:** The engines are small (50-120 lines each) and tightly coupled to the orchestrator. A subdirectory adds import ceremony for no benefit. Each engine file is self-contained — struct, constructor, `Search()`/`News()` method, and any helper functions.

### 2. No parse/ package

**Decision:** HTML parsing helpers stay private and co-located with each engine.

**Why:** Each engine has its own HTML structure and parsing quirks. Extracting shared "parse helpers" would create false abstractions — the parsing code between DuckDuckGo and Brave has nothing in common. Functions are small, private, and belong next to their callers.

### 3. HTTPDoer interface over concrete *http.Client

**Decision:** Engines accept `HTTPDoer` interface (`Do(*http.Request) (*http.Response, error)`) instead of `*http.Client`.

**Why:** This lets engines work with both `*http.Client` (in tests, using httptest) and `*httpclient.Client` (in production, with UA rotation + rate limiting). Engines stay testable without mocking infrastructure.

### 4. Multi-backend fallback over exponential backoff

**Decision:** No `cenkalti/backoff` dependency. Retry strategy is multi-backend fallback, not per-request retries.

**Why:** This is a CLI tool, not a long-running service. When a user runs a search, they want results in seconds, not after a backoff sequence. If DuckDuckGo fails, we immediately try Brave, Mojeek, etc. The health tracker prevents repeatedly hitting a broken backend (exponential cooldown: 30s → 5min). This gives better UX than retrying the same failing backend with increasing delays.

### 5. Concurrent dispatch with errgroup

**Decision:** All backends in the auto set run concurrently via `errgroup`, not sequentially.

**Why:** With 4 default backends, sequential dispatch means total latency = sum of all backend latencies. Concurrent dispatch means latency = max(backend latencies). Results are collected with a mutex, then sorted by engine priority before ranking to maintain deterministic output.

### 6. Separate fetch client with SSRF protection

**Decision:** `fetchClient()` (for `obk websearch fetch`) uses a raw `*http.Client` with SSRF guards. `httpClient()` (for search engines) uses `*httpclient.Client` without SSRF guards.

**Why:** Search engines hit known external hosts (duckduckgo.com, bing.com, etc.) — SSRF isn't a concern. But `fetch` takes arbitrary user-provided URLs, so it must resolve DNS and block private IPs (loopback, RFC1918, link-local) to prevent SSRF. It also pins to the first resolved IP to prevent DNS rebinding (TOCTOU).

### 7. UA rotation per client instance, not per request

**Decision:** A random browser profile (Chrome/Firefox/Safari/Edge) is selected when `httpclient.Client` is created and reused for all requests from that client.

**Why:** A real browser sends the same UA across a session. Rotating per-request looks suspicious to anti-bot systems. The client is created once per `WebSearch` instance and reused across searches.

### 8. Auto backend set: DDG + Brave + Mojeek + Wikipedia

**Decision:** The default `auto` set uses DuckDuckGo, Brave, Mojeek, and Wikipedia. Yahoo, Yandex, Google, and Bing are opt-in only via `--backend <name>`.

**Why:**
- **DuckDuckGo** — Most reliable scraping target. No-JS HTML endpoint.
- **Brave** — Good quality, less aggressive anti-bot than Google.
- **Mojeek** — Very permissive, independent index (not Bing/Google derivative).
- **Wikipedia** — JSON API (no scraping needed), always high-quality for factual queries.
- **Google** — Most aggressive anti-bot, CAPTCHAs likely. Opt-in only.
- **Bing** — Disabled in ddgs too. Opt-in only.
- **Yahoo/Yandex** — Redirect URL unwrapping adds fragility. Opt-in only.

Users can override via config (`websearch.backends` list) or `--backend` flag.

### 9. Ranking: frequency + token scoring + Wikipedia priority

**Decision:** Results are ranked by: multi-backend appearance bonus, query token scoring (title weight 2x, snippet weight 1x), Wikipedia +10 bonus. Stable sort preserves original order for equal scores.

**Why:** Simple, predictable, no ML. A result appearing from 3 backends is likely more relevant than one from 1 backend. Title matches matter more than snippet matches. Wikipedia is almost always the best single result for factual queries — the +10 bonus ensures it surfaces first without suppressing other results.

### 10. Cache key includes page number

**Decision:** Cache key = `sha256(query|category|backend|region|timeLimit|page)`.

**Why:** Without page in the key, searching "golang" page 1 then "golang" page 2 returns page 1's cached results. Learned this from a bug caught in review.

### 11. Best-effort caching and history, not transactional

**Decision:** `putSearchCache`, `putFetchCache`, and `putSearchHistory` log warnings on failure but don't propagate errors.

**Why:** Cache and history are convenience features. A failed cache write shouldn't make a successful search return an error. The slog.Warn ensures failures are visible for debugging.

### 12. Bing URL unwrapping via base64

**Decision:** Bing wraps result URLs in `/ck/a?u=<encoded>` redirects. We decode inline: strip first 2 chars of the `u` param, base64url decode the rest.

**Why:** Following Bing's real redirect URL isn't reliable (may require cookies/sessions). The base64 encoding scheme was reverse-engineered from ddgs and is stable.

---

## What We Chose Not to Build

| Feature | Reason |
|---------|--------|
| Exponential backoff (`cenkalti/backoff`) | Multi-backend fallback is the retry strategy for a CLI tool |
| `SearchError` structured error type | Plain `fmt.Errorf` is sufficient; errors flow through cobra's `RunE` |
| `--proxy` CLI flag / `WEBSEARCH_PROXY` env var | Proxy works via config.yaml; CLI flag adds complexity for a rarely-used feature |
| Session recovery (401/403 retry) | Multi-backend fallback handles this implicitly |
| Rate limiter eviction | Not needed for CLI (process is short-lived); TODO left for library use |
| Daemon cache warming | Deferred — low value until usage patterns are established |

---

## Backend Reference

| Backend | Method | Priority | Auto | Notes |
|---------|--------|----------|------|-------|
| DuckDuckGo | POST `html.duckduckgo.com/html/` | 1 | Yes | Most reliable. No-JS endpoint. Max 499 char query. |
| Brave | GET `search.brave.com/search` | 1 | Yes | Good quality, moderate anti-bot. |
| Mojeek | GET `www.mojeek.com/search` | 1 | Yes | Very permissive. Independent index. |
| Wikipedia | GET `en.wikipedia.org/w/api.php` | 2 | Yes | JSON API. +10 ranking bonus. Filters disambiguation. |
| Yahoo | GET `search.yahoo.com/search` | 1 | No | Requires redirect URL unwrapping. |
| Yandex | GET `yandex.com/search/site/` | 1 | No | Uses random searchid. |
| Google | GET `www.google.com/search` | 0 | No | Most aggressive anti-bot. Lowest priority. |
| Bing | GET `www.bing.com/search` | 0 | No | base64 URL unwrapping. Ad filtering. |

News backends: DuckDuckGo (VQD token + JSON API), Yahoo (HTML scraping).

---

## Security

- **SSRF protection on fetch**: DNS resolution + private IP blocking + IP pinning (anti-DNS-rebinding)
- **No SSRF on search**: Engines only hit known external hosts
- **SQL injection**: All queries use parameterized placeholders (`db.Rebind("... ?")`)
- **Query length limit**: 2000 chars max to prevent abuse
- **Response body limit**: 10MB hard cap on fetch, body size limits on news/VQD responses

---

## Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/PuerkitoBio/goquery` | HTML parsing with CSS selectors |
| `golang.org/x/time/rate` | Per-host token bucket rate limiting |
| `golang.org/x/sync/errgroup` | Concurrent backend dispatch |
| `github.com/JohannesKaufmann/html-to-markdown` | HTML → Markdown for fetch |
| `github.com/refraction-networking/utls` | TLS fingerprint impersonation (via internal/browser) |

---

## Prior Art

| Project | Language | Gap |
|---------|----------|-----|
| [ddgs](https://github.com/deedy5/ddgs) | Python | Python-only. Uses `primp` (Rust) for TLS. |
| [SearXNG](https://github.com/searxng/searxng) | Python | Requires Docker. Server-based. |
| [Djarvur/ddg-search](https://github.com/Djarvur/ddg-search) | Go | DDG only, no multi-backend. |

This fills the gap: a Go data source with multi-backend search, skills integration, and agent-first CLI design.
