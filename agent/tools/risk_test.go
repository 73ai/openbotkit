package tools

import "testing"

func TestRiskLevel_Constants(t *testing.T) {
	if RiskLow >= RiskMedium {
		t.Error("RiskLow should be less than RiskMedium")
	}
	if RiskMedium >= RiskHigh {
		t.Error("RiskMedium should be less than RiskHigh")
	}
}
