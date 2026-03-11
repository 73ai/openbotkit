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

	// GWS section — only if gws_execute tool is registered.
	if reg.Has("gws_execute") {
		b.WriteString(`
## Google Workspace
Use the gws_execute tool for all Google Workspace operations (Calendar, Drive, Docs, Sheets, Tasks, Contacts).
Do NOT use bash to run gws commands — they will be rejected. Always use gws_execute instead.
The tool handles authentication, scope checks, and approval for write operations automatically.
When a tool result includes "user notified", keep your response brief — the user already got the details.
`)
	}

	// Delegate section — only if delegate_task tool is registered.
	if reg.Has("delegate_task") {
		b.WriteString(`
## Task Delegation
Use delegate_task for complex tasks: research, analysis, code generation.
For multi-step workflows, provide steps in the spec — they execute as a single agent run.
Set async=true for tasks taking more than a minute. You'll be notified of progress periodically.
Use check_task to retrieve results, then deliver them using other tools (gws_execute for Google Docs, slack_send for Slack, etc.).
Example workflow: delegate research → check_task → gws_execute to create doc → slack_send to share link.
`)
	}

	// Slack section — only if slack tools are registered.
	if reg.Has("slack_search") {
		b.WriteString(`
## Slack
Use the slack_search, slack_read_channel, and slack_read_thread tools for reading Slack content.
Use slack_send, slack_edit, and slack_react for write operations (these require user approval).
Channel references accept: #name, C-ID, or Slack archive URL. User references accept: @handle, U-ID, or email.
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
