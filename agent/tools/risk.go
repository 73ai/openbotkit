package tools

// RiskLevel classifies the sensitivity of a tool action.
type RiskLevel int

const (
	RiskLow    RiskLevel = iota // auto-approve, notify only
	RiskMedium                  // standard approval (current behavior)
	RiskHigh                    // enhanced approval with full preview
)

