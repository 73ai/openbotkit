# Use Case: Deep Research

## Scenario 1 — Subagent multi-faceted web research
User asks for a comparison of vector databases. The parent agent
delegates to a subagent which performs multiple web searches, fetches
pages, and returns a synthesized comparison.

### Steps
1. User asks to compare Pinecone, Weaviate, and Qdrant
   -> Agent delegates to subagent -> subagent uses web_search + web_fetch
   -> parent synthesizes and delivers a coherent comparison

## Scenario 2 — External agent delegation (delegate_task)
User asks for research on WebAssembly server-side runtimes. The parent
delegates to an external AI CLI agent via delegate_task.

### Steps
1. User asks about WebAssembly server-side runtimes
   -> Agent delegates via delegate_task to external agent (claude/gemini)
   -> reads result file -> delivers a synthesized summary
