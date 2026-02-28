package parser

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// "Total: $137.18 USD" — always use this for GitHub
	reGitHubTotal = regexp.MustCompile(`(?i)Total:\s*\$([\d,]+\.?\d*)\s*USD`)
	// "Charged to: Visa (4*** **** **** 1646)"
	reGitHubCharged = regexp.MustCompile(`(?i)charged to:\s*(\w+)\s*\([*\d\s]*?(\d{4})\)`)
)

// ParseGitHubEmail extracts amount and payment method from GitHub payment receipt emails.
// GitHub receipts list individual line items then a "Total: $X.XX USD" line — we always
// extract the total to avoid grabbing a line-item amount.
func ParseGitHubEmail(subject, body, htmlBody string) ParsedEmail {
	var pe ParsedEmail

	searchText := body

	if m := reGitHubTotal.FindStringSubmatch(searchText); len(m) == 2 {
		pe.Amount = "$" + strings.ReplaceAll(m[1], ",", "")
	}

	if m := reGitHubCharged.FindStringSubmatch(searchText); len(m) == 3 {
		pe.PaymentMethod = fmt.Sprintf("Paid via %s ending %s", m[1], m[2])
	}

	return pe
}
