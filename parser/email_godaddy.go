package parser

import (
	"regexp"
	"strings"
)

var (
	// "Total:" followed (possibly on next line) by "₹ 749.00"
	reGoDaddyTotal = regexp.MustCompile(`(?i)Total:\s*₹\s*([\d,]+\.?\d*)`)
	// Fallback: first "₹ X" in HTML
	reGoDaddyFirstRupee = regexp.MustCompile(`₹\s*([\d,]+\.?\d*)`)
)

// ParseGoDaddyEmail extracts amount from GoDaddy order confirmation emails.
// GoDaddy emails are HTML-only (empty plain text body) with "Total: ₹ 749.00".
func ParseGoDaddyEmail(subject, body, htmlBody string) ParsedEmail {
	var pe ParsedEmail

	// GoDaddy emails are HTML-only
	if m := reGoDaddyTotal.FindStringSubmatch(htmlBody); len(m) == 2 {
		pe.Amount = "₹" + strings.ReplaceAll(m[1], ",", "")
	} else if m := reGoDaddyFirstRupee.FindStringSubmatch(htmlBody); len(m) == 2 {
		pe.Amount = "₹" + strings.ReplaceAll(m[1], ",", "")
	}

	return pe
}
