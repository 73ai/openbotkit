package parser

import (
	"regexp"
	"strings"
)

var (
	// "Invoiced amount\n₹960.00"
	reFreepikInvoicedAmt = regexp.MustCompile(`(?i)Invoiced amount\s*₹\s*([\d,]+\.?\d*)`)
	// Fallback: first currency amount in body
	reFreepikFirstCurrency = regexp.MustCompile(`([₹$€])\s*([\d,]+\.?\d*)`)
)

// ParseFreepikEmail extracts amount from Freepik invoice emails.
// Format: "Invoiced amount ₹960.00" in plain text body.
func ParseFreepikEmail(subject, body, htmlBody string) ParsedEmail {
	var pe ParsedEmail

	if m := reFreepikInvoicedAmt.FindStringSubmatch(body); len(m) == 2 {
		pe.Amount = "₹" + strings.ReplaceAll(m[1], ",", "")
	} else if m := reFreepikFirstCurrency.FindStringSubmatch(body); len(m) == 3 {
		pe.Amount = m[1] + strings.ReplaceAll(m[2], ",", "")
	}

	return pe
}
