package tools

// RiskLevel classifies the sensitivity of a tool action.
type RiskLevel int

const (
	RiskLow    RiskLevel = iota // auto-approve, notify only
	RiskMedium                  // standard approval (current behavior)
	RiskHigh                    // enhanced approval with full preview
)

// toolRiskLevels maps tool names to their default risk level.
// Tools not listed default to RiskMedium.
var toolRiskLevels = map[string]RiskLevel{
	"slack_react":   RiskLow,
	"slack_send":    RiskMedium,
	"slack_edit":    RiskMedium,
	"gws_execute":   RiskHigh,
	"delegate_task": RiskHigh,
}

// ToolRiskLevel returns the risk level for a tool. Defaults to RiskMedium.
func ToolRiskLevel(toolName string) RiskLevel {
	if level, ok := toolRiskLevels[toolName]; ok {
		return level
	}
	return RiskMedium
}
