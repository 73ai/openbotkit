package provider

var contextWindows = map[string]int{
	"claude-opus-4-6":      200000,
	"claude-sonnet-4-6":    200000,
	"claude-haiku-4-5":     200000,
	"gpt-4o":               128000,
	"gpt-4o-mini":          128000,
	"gemini-2.5-pro":       1048576,
	"gemini-2.5-flash":     1048576,
	"gemini-2.0-flash-lite": 1048576,
}

func DefaultContextWindow(model string) int {
	return contextWindows[model]
}
