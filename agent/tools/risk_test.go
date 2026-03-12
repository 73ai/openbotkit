package tools

import "testing"

func TestToolRiskLevel(t *testing.T) {
	cases := []struct {
		tool string
		want RiskLevel
	}{
		{"slack_react", RiskLow},
		{"slack_send", RiskMedium},
		{"slack_edit", RiskMedium},
		{"gws_execute", RiskHigh},
		{"delegate_task", RiskHigh},
		{"unknown_tool", RiskMedium},
		{"bash", RiskMedium},
	}
	for _, tc := range cases {
		if got := ToolRiskLevel(tc.tool); got != tc.want {
			t.Errorf("ToolRiskLevel(%q) = %d, want %d", tc.tool, got, tc.want)
		}
	}
}
