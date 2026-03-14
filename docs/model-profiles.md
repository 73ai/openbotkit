# Model Profiles & Tier System

## Overview

The model tier system routes LLM requests to different models based on task complexity. Profiles are preset tier→model mappings that simplify configuration.

## Tiers

| Tier | Purpose | Example tasks |
|------|---------|---------------|
| Default | Main conversation, skill execution | Chat, multi-turn tool calling |
| Complex | Tasks requiring strongest reasoning | (falls back to default if not set) |
| Fast | Latency-sensitive background tasks | Context compaction, memory extraction, web fetch summarization |
| Nano | Trivially simple tasks | Tool acknowledgments, memory reconciliation |

### Cascade fallback

When a tier's model isn't configured, it cascades:

- **nano** → fast → default
- **complex** → default
- **fast** → default

This means existing configs without `nano` work unchanged — nano requests fall through to fast or default.

## Profiles

Profiles preset all four tiers based on monthly budget. User only needs 2 API keys (OpenRouter + Gemini).

| Profile | ~Cost/mo | Default | Complex | Fast | Nano |
|---------|----------|---------|---------|------|------|
| Starter | $20 | Mistral Medium 3.1 (OR) | Mistral Medium 3.1 (OR) | Gemini 2.0 Flash-Lite | Gemini 2.0 Flash-Lite |
| Standard | $50 | Claude Haiku 4.5 (OR) | Claude Sonnet 4.6 (OR) | Gemini 2.0 Flash-Lite | Gemini 2.0 Flash-Lite |
| Premium | $100 | Claude Sonnet 4.6 (OR) | Claude Opus 4.6 (OR) | Claude Haiku 4.5 (OR) | Gemini 2.0 Flash-Lite |

*OR = via OpenRouter*

### Why OpenRouter for default/complex/fast

OpenRouter gives access to Claude, GPT, Gemini, Mistral, and open-source models via a single API key. This avoids requiring users to sign up for 3-4 separate provider accounts. The tradeoff is a small markup on per-token pricing and an extra network hop.

### Why Gemini direct (not via OpenRouter) for nano/fast

Nano and fast tiers are latency-sensitive — they run during active user conversations (tool acks, compaction). Routing through OpenRouter adds ~50-100ms of extra latency per request. Using Google AI Studio directly for Gemini 2.0 Flash-Lite avoids this double-hop. The free tier (1500 RPD) covers typical nano/fast usage.

### Why not Groq for nano

We initially planned to use Groq (free Llama 3.1 8B) for nano tasks. We switched to Gemini 2.0 Flash-Lite because:

1. **Fewer API keys** — profiles already need Gemini for fast tier, so nano reuses the same key
2. **Better instruction following** — Gemini Flash-Lite handles JSON output (tool ack decisions) more reliably than Llama 8B
3. **Simpler architecture** — two providers (OpenRouter + Gemini) instead of three

Groq is still registered as a provider and available for custom configurations.

## New providers

### Groq (`provider/groq`)

Groq's API is OpenAI-compatible. The provider is a thin wrapper (~15 lines) that reuses `openai.New()` with `https://api.groq.com/openai` as the base URL. Env var: `GROQ_API_KEY`.

### OpenRouter (`provider/openrouter`)

Same pattern as Groq — reuses OpenAI provider with `https://openrouter.ai/api` as the base URL. Model specs use nested slashes: `openrouter/anthropic/claude-sonnet-4-6`. `ParseModelSpec` already handles this correctly via `strings.SplitN(spec, "/", 2)`. Env var: `OPENROUTER_API_KEY`.

## Task→Tier assignments

| Task | Tier | Rationale |
|------|------|-----------|
| Main conversation | Default | Needs strong instruction following, 128k+ context for skills |
| Tool acknowledgments | Nano | Simple JSON output, 3s timeout, latency-critical |
| Memory reconciliation | Nano | Compare-and-merge, structured output |
| Memory extraction | Fast | Needs judgment to identify salient memories |
| Context compaction | Fast | Summarization quality matters |
| Web fetch summarization | Fast | Summarization quality matters |

## CLI setup flow

`obk setup models` now starts with profile selection:

```
How would you like to configure models?
→ Starter (~$20/mo)
→ Standard (~$50/mo)
→ Premium (~$100/mo)
→ Custom (manual configuration)
```

**Profile flow:** show tier mapping → prompt for required API keys → validate → write config.

**Custom flow:** select providers (now includes OpenRouter + Groq) → configure auth → select models → assign tiers (now includes nano) → validate.

The default model is validated for 128k+ context window (required for the skill system).

## Config format

```yaml
models:
  profile: standard
  default: openrouter/anthropic/claude-haiku-4-5
  complex: openrouter/anthropic/claude-sonnet-4-6
  fast: gemini/gemini-2.0-flash-lite
  nano: gemini/gemini-2.0-flash-lite
  providers:
    openrouter:
      api_key_ref: "keychain:obk/openrouter"
    gemini:
      api_key_ref: "keychain:obk/gemini"
```

The `profile` field is informational — the router only reads the four tier fields. Profiles are config presets, not a runtime concept.

## Design decisions

1. **Profiles are pure config presets.** The Router doesn't know about profiles. A profile just pre-fills the 4 tier fields. This keeps the routing architecture simple.

2. **Groq and OpenRouter reuse OpenAI provider via composition.** Both are OpenAI-compatible APIs. Each provider is ~15 lines wrapping `openai.New()` with a different base URL. No code duplication, no new HTTP/JSON logic.

3. **Nano cascades through fast before default.** This mirrors how complex cascades to default, but adds an intermediate step. A config with only `default` set still works for all four tiers.

4. **Default model must support skills.** The skill system injects SKILL.md into context and requires multi-turn tool calling. This needs 128k+ context and strong instruction following. The CLI warns if the default model has insufficient context.
