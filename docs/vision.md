# Vision

Personal assistants are going to be one of the most useful applications of AI. They'll handle email, messages, scheduling, documents, reminders — the operational overhead that eats hours every week. The economic value is real and obvious.

But here's the problem: most personal assistant projects treat safety as an afterthought. They give an AI agent access to your email, your messages, your files, and let it run autonomously in the background. It hallucates. It acts on wrong assumptions. It sends things you didn't mean to send. And you can't see what it's doing because it's a black box running on someone else's servers.

OpenBotKit exists because we believe you can build a personal assistant that delivers real value without compromising on safety. Safety first — not safety later, not safety as a checkbox.

## Why we started this

This project was inspired by watching projects like OpenClaw push the boundaries of what AI agents can do. The capability was impressive. The safety story was not. Giving an autonomous agent the keys to your email and messages with no approval flow and no transparency is a disaster waiting to happen.

We wanted to prove that safety and usefulness are not at odds. You can build the world's best personal assistant *and* make it the safest one. Quality software, responsible defaults, real value.

## What we believe

### Safety is not a tradeoff

Every action the assistant takes that affects the outside world requires explicit user approval. If the agent wants to send an email, it sends you a message on Telegram with approve and deny buttons. You review the draft, you press approve, then it sends. Not before.

This isn't a limitation — it's the whole point. An assistant that autonomously floods someone's inbox from your account isn't providing value, it's creating liability. Whatever we do, we do responsibly. We don't screw the user.

### Transparency over magic

You should be able to see exactly what the assistant is doing at every step. Every database query, every API call, every message it wants to send. No hidden agent loops running behind your back. No opaque "AI magic" that you can't inspect.

The assistant's skills are plain text files with SQL patterns and CLI commands. You can read any skill in 30 seconds. The data lives in SQLite databases you can query yourself with `sqlite3`. There is no hidden complexity. If you don't understand what the assistant is doing, that's a bug.

### Local first

Your data syncs into SQLite databases on your machine. Nothing leaves your device unless you explicitly send it. OpenBotKit connects directly to Gmail's API, WhatsApp's protocol, and Apple Notes on your Mac. No relay server, no cloud middleware, no third-party backend sitting between you and your data.

For users who want access from anywhere, there's an option to deploy on a remote server. But local is the default, and the remote deployment follows the same safety principles — your local machine acts as the operator for the remote instance.

### A kit, not a product

OpenBotKit is a box of ingredients, not a pre-made meal. We provide data connectors, a sync engine, a local database, a CLI, an agent loop, and assistant scaffolding. You put it together the way you want.

- Bring your own agent — Claude Code, our built-in agent, or any agent that supports tool use.
- Pick your integrations from the registry — Gmail, WhatsApp, Apple Notes, Google Workspace, or build your own.
- Bring third-party skills or write your own.
- Interface through any channel — CLI, Telegram, Element.io, or build a custom client with our API.
- Deploy however you want — on your laptop, in Docker, on Kubernetes.

For non-technical users, it works out of the box as a product. But it's designed as a kit. We're the ingredient provider, not the chef. You take the components, modify them, combine them, and build something that fits your life.

### No slop

The AI assistant space is full of bloated agent frameworks, magic tool-calling you can't inspect, and 200-dependency packages that break every other week.

OpenBotKit is a single Go binary. The skills are readable text files. The data is queryable SQLite. The code is straightforward Go with tests. We don't add abstraction layers unless they earn their complexity. We don't add dependencies unless they're necessary. If three lines of simple code solve the problem, we don't write a framework.

## What we're building toward

A personal assistant that produces real economic value — saving users hours every week on email triage, message management, scheduling, document search, reminders, and routine tasks. Accessible from your phone, your terminal, or wherever you need it.

The assistant should be something you actually trust. Trust because you can see what it's doing. Trust because it asks before acting. Trust because your data never leaves your machine unless you say so. Trust because the code is simple enough to read and understand.

That's the bar. Safety and quality, no compromises.
