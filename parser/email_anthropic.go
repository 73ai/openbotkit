package parser

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// "Amount paid $200" or "Amount paid $118.00"
	reAnthropicAmountPaid = regexp.MustCompile(`(?i)Amount paid\s+\$([\d,]+\.?\d*)`)
	// "Payment method - 1646" or "Payment method Link"
	reAnthropicPayMethod = regexp.MustCompile(`(?i)payment method\s*[-–]?\s*(\d{4}|Link)`)
	// Fallback: first "$X.XX" in body
	reAnthropicFirstDollar = regexp.MustCompile(`\$([\d,]+\.?\d*)`)
)

// ParseAnthropicEmail extracts amount and payment method from Anthropic receipt emails.
// Format: Stripe-generated receipts with "Amount paid $X" and "Payment method - 1646".
func ParseAnthropicEmail(subject, body, htmlBody string) ParsedEmail {
	var pe ParsedEmail

	searchText := body + " " + subject

	// Prefer "Amount paid $X"
	if m := reAnthropicAmountPaid.FindStringSubmatch(searchText); len(m) == 2 {
		pe.Amount = "$" + strings.ReplaceAll(m[1], ",", "")
	} else if m := reAnthropicFirstDollar.FindStringSubmatch(searchText); len(m) == 2 {
		pe.Amount = "$" + strings.ReplaceAll(m[1], ",", "")
	}

	// Payment method
	if m := reAnthropicPayMethod.FindStringSubmatch(searchText); len(m) == 2 {
		if m[1] == "Link" {
			pe.PaymentMethod = "Paid via Stripe Link"
		} else {
			pe.PaymentMethod = fmt.Sprintf("Paid via card ending %s", m[1])
		}
	}

	return pe
}
