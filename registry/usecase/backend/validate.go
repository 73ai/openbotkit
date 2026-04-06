package main

import "fmt"

var validDomains = map[string]bool{
	"Customer Support": true,
	"Finance":          true,
	"Productivity":     true,
	"Trust & Safety":   true,
	"Sales":            true,
	"Healthcare":       true,
	"Engineering":      true,
	"Legal":            true,
	"Marketing":        true,
	"HR":               true,
	"Operations":       true,
}

var validRiskLevels = map[string]bool{
	"low": true, "medium": true, "high": true,
}

var validROILevels = map[string]bool{
	"low": true, "medium": true, "high": true,
}

var validStatuses = map[string]bool{
	"draft": true, "published": true, "archived": true,
}

var validImplStatuses = map[string]bool{
	"evaluating": true, "pilot": true, "production": true, "deprecated": true,
}

var validVisibilities = map[string]bool{
	"public": true, "private": true,
}

func validateUseCaseRequest(req *useCaseRequest) error {
	if req.Title == "" {
		return fmt.Errorf("title is required")
	}
	if len(req.Title) > 200 {
		return fmt.Errorf("title must be under 200 characters")
	}
	if req.Description == "" {
		return fmt.Errorf("description is required")
	}
	if len(req.Description) > 5000 {
		return fmt.Errorf("description must be under 5000 characters")
	}
	if req.Domain == "" {
		return fmt.Errorf("domain is required")
	}
	if !validDomains[req.Domain] {
		return fmt.Errorf("invalid domain: %s", req.Domain)
	}
	if req.RiskLevel != "" && !validRiskLevels[req.RiskLevel] {
		return fmt.Errorf("invalid risk level: %s", req.RiskLevel)
	}
	if req.ROIPotential != "" && !validROILevels[req.ROIPotential] {
		return fmt.Errorf("invalid ROI potential: %s", req.ROIPotential)
	}
	if req.Status != "" && !validStatuses[req.Status] {
		return fmt.Errorf("invalid status: %s", req.Status)
	}
	if req.ImplStatus != "" && !validImplStatuses[req.ImplStatus] {
		return fmt.Errorf("invalid implementation status: %s", req.ImplStatus)
	}
	if req.Visibility != "" && !validVisibilities[req.Visibility] {
		return fmt.Errorf("invalid visibility: %s", req.Visibility)
	}
	return nil
}
