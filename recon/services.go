package recon

import (
	"time"

	"github.com/priyanshujain/reimbursement/parser"
)

// SurchargeRule defines a pattern-based surcharge to look for on reconciled transactions.
// For example, a 1% "Rent Transaction Fee" followed by 18% GST on WeWork payments.
type SurchargeRule struct {
	Pattern    string  // statement description pattern to search for (e.g. "Rent Transaction Fee")
	Percentage float64 // surcharge as fraction of parent amount (e.g. 0.01 for 1%)
	MaxDays    int     // max days after parent transaction to search
	GSTRate    float64 // GST rate applied on the surcharge (e.g. 0.18 for 18%)
}

// Service defines how to identify a reimbursable service across data sources.
type Service struct {
	Name             string
	AfterDate        time.Time              // exclusive: last reimbursed date (transactions on this date are excluded)
	DestPatterns     []string               // case-insensitive LIKE patterns for destination statement descriptions
	EmailFroms       []string               // substrings to match in email From field (any = OR)
	EmailSubjects    []string               // substrings that must ALL appear in Subject (AND)
	ExcludeSubject   []string               // substrings that must NOT appear in Subject (exclusion filter)
	DashboardInvoice string                 // provider name for dashboard invoices (e.g. "freepik")
	OfflineDir       string                 // path to folder with scanned invoices (auto-reconciled, no proof needed)
	EmailParser      parser.EmailParserFunc // per-provider email amount/payment extraction
	Optional         bool
	Surcharges       []SurchargeRule // surcharge rules to apply to reconciled transactions
}
