# OpenBotKit — Da Nang AI Meetup
## Slide Content (March 15, 2026)

---

### Slide 1: Title

**OpenBotKit**
Your AI assistant. Your rules.

_Priyanshu Jain — Da Nang AI Meetup_

---

### Slide 2: AI Agents Are Powerful

Claude Code showed us what's possible:
- Read your emails, summarize them
- Send messages on your behalf
- Search the web, schedule meetings
- Handle tasks that used to take hours

**AI agents can do real work.**

---

### Slide 3: But There's a Problem

What happens when the agent:
- Sends an email you didn't approve?
- Forwards a private conversation to the wrong person?
- Spams your WhatsApp contacts and gets your account blocked?
- Runs in the background on someone else's server with your data?

**Power without control is not useful — it's dangerous.**

---

### Slide 4: Real Examples

- WhatsApp bans accounts that use unauthorized automation
- AI agents hallucinate — they draft wrong replies, misread context
- Your emails, messages, and contacts are on someone else's cloud
- You can't see what the agent is doing — it's a black box

_This isn't hypothetical. This is happening today._

---

### Slide 5: What If There Was a Better Way?

**OpenBotKit** — an open-source personal assistant that:

1. **Runs on YOUR machine** — your data never leaves your device
2. **Always asks before acting** — every message, every email needs your OK
3. **You can see everything** — no hidden loops, no black box

---

### Slide 6: How It Works

[Excalidraw diagram here]

```
You ←→ Telegram
         ↓
    OpenBotKit (on your laptop)
         ↓
   ┌─────┼─────┐
   ↓     ↓     ↓
 Gmail WhatsApp Web Search
   ↓     ↓     ↓
   SQLite databases (local)
```

Everything stays on your machine.
The agent connects directly to Gmail API, WhatsApp protocol.
No middleman. No cloud relay.

---

### Slide 7: The Safety Promise

When the agent wants to send a message or email:

1. It drafts the message
2. Sends you a preview on Telegram
3. You see **Approve** / **Deny** buttons
4. Only after you tap Approve → it sends

**This is enforced in code, not by asking the AI nicely.**
The AI cannot skip this step, no matter what.

---

### Slide 8: Demo — WhatsApp

_[Live demo / Video]_

- "What did [friend] say on WhatsApp today?"
- Bot reads messages, gives a summary
- "Tell them I'll be 10 minutes late"
- Telegram shows: preview + Approve/Deny
- Tap Approve → message sent

**No account blocked. No spam. Because it asks first.**

---

### Slide 9: Demo — Gmail

_[Live demo / Video]_

- "What emails did I get today?"
- Bot summarizes recent emails
- "Reply to [person] — thanks, I'll be there Sunday"
- Telegram shows: full draft + Approve/Deny
- Tap Approve → email sent

**You stay in control. Always.**

---

### Slide 10: Why Open Source?

- You can read every line of code
- You can see exactly what the assistant does with your data
- You can modify it, extend it, or remove what you don't need
- No vendor lock-in, no subscription, no terms of service surprises

**Trust through transparency.**

---

### Slide 11: Try It

github.com/73ai/openbotkit

[QR code]

- Single Go binary — one install
- Connect Gmail, WhatsApp, or both
- Talk to your assistant via Telegram or terminal
- Everything local. Everything safe.

**Questions?**

---

## Speaker Notes

### On WhatsApp angle
- Many people in Da Nang use WhatsApp daily. Some have had accounts blocked by unauthorized bots. Lead with this — it's relatable.
- Emphasize: OpenBotKit uses the WhatsApp protocol properly and asks before sending. This is why accounts don't get blocked.

### On safety during Q&A
- If asked "what if someone just approves everything?" → Mention rubber-stamp detection (warns if 5+ approvals in 30 seconds)
- If asked "can the AI bypass the approval?" → No. It's enforced in code (GuardedAction), not by prompting. The LLM literally cannot skip it.
- If asked about prompt injection → Honest answer: unsolved problem industry-wide, but we have 8 defense layers. Show docs/safety.md for the deep dive.

### On "why not just use ChatGPT/Claude?"
- Those are chat interfaces. They can't send emails from YOUR account, read YOUR WhatsApp, or act on YOUR behalf with approval gates.
- OpenBotKit is not competing with ChatGPT — it's the layer that connects AI to your personal data, safely.

### Demo timing
- WhatsApp demo: ~2 minutes
- Gmail demo: ~2 minutes
- Keep it tight. Don't show more than 2 features. Depth > breadth for a 10-minute talk.
