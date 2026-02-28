package parser

import (
	"regexp"
	"strings"
)

// "for INR 1593.00" or "for INR 849.6"
var reWeWorkINR = regexp.MustCompile(`(?i)for INR\s+([\d,]+\.?\d*)`)

// ParseWeWorkEmail extracts amount from WeWork/myHQ invoice emails.
// Format: "Your invoice is issued on 2025-07-17 for INR 1593.00"
func ParseWeWorkEmail(subject, body, htmlBody string) ParsedEmail {
	var pe ParsedEmail

	if m := reWeWorkINR.FindStringSubmatch(body); len(m) == 2 {
		pe.Amount = "₹" + strings.ReplaceAll(m[1], ",", "")
	}

	return pe
}
