package tools

import (
	"fmt"
	"strings"

	"github.com/priyanshujain/openbotkit/internal/skills"
	"github.com/priyanshujain/openbotkit/provider"
)

// BuildBaseSystemPrompt generates the shared portion of the system prompt
// from the given registry. The tool list is derived from the registry
// so it can never drift out of sync.
//
// Callers should prepend their own identity line (e.g. "You are...via Telegram")
// and append any site-specific sections (e.g. user memories).
func BuildBaseSystemPrompt(reg *Registry) string {
	var b strings.Builder

	// Tool list — auto-generated from registry.
	b.WriteString("\n## Tools\n")
	b.WriteString("Available: ")
	b.WriteString(strings.Join(reg.ToolNames(), ", "))
	b.WriteString(".\n")
	b.WriteString("Tool names are case-sensitive. Call tools exactly as listed.\n")

	// Tool usage rules.
	b.WriteString(`
Rules:
- ALWAYS use tools to perform actions. Never say you will do something without calling the tool.
- Never predict or claim results before receiving them. Wait for tool output.
- Do not narrate routine tool calls — just call the tool. Only explain when the step is non-obvious or the user asked for details.
- If a tool call fails, analyze the error before retrying with a different approach.
`)

	// Sub-agents section — only if the subagent tool is registered.
	if reg.Has("subagent") {
		b.WriteString(`
## Sub-agents
Use the subagent tool to delegate self-contained sub-tasks that don't need your conversation history.
Good uses: independent research, file operations, or multi-step tasks that can run in isolation.
Do not use subagent for simple single-tool calls — just call the tool directly.
The sub-agent has its own tools (bash, file ops, skills) but cannot spawn further sub-agents.
`)
	}

	// Skills section.
	b.WriteString(`
## Skills
Before replying to domain-specific requests (email, WhatsApp, memories, notes, etc.):
1. Scan the "Available skills" list below for matching skill names
2. Use load_skills to read the skill's instructions
3. Use bash to run the commands from those instructions
4. If the request spans multiple domains, load and use ALL relevant skills
5. If no skill matches, use search_skills to discover one by keyword
`)

	idx, err := skills.LoadIndex()
	if err == nil && len(idx.Skills) > 0 {
		b.WriteString("\nAvailable skills:\n")
		for _, s := range idx.Skills {
			fmt.Fprintf(&b, "- %s: %s\n", s.Name, s.Description)
		}
	}

	return b.String()
}

// BuildSystemBlocks returns a structured system prompt split into
// a cacheable base block (identity + tool instructions) and an optional
// extras block (memories, channel rules) that may change between sessions.
func BuildSystemBlocks(identity string, reg *Registry, extras ...string) []provider.SystemBlock {
	base := identity + BuildBaseSystemPrompt(reg)
	blocks := []provider.SystemBlock{
		{Text: base, CacheControl: &provider.CacheControl{Type: "ephemeral"}},
	}
	if extra := strings.Join(extras, ""); extra != "" {
		blocks = append(blocks, provider.SystemBlock{Text: extra})
	}
	return blocks
}
