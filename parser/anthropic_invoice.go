package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	reAnthropicInvoiceNum  = regexp.MustCompile(`Invoice number\s+([\w-]+)`)
	reAnthropicDate        = regexp.MustCompile(`Date of issue\s+(\w+ \d{1,2}, \d{4})`)
	reAnthropicAmountDue   = regexp.MustCompile(`\$([\d,]+\.?\d*)\s+USD due`)
	reAnthropicDescription = regexp.MustCompile(`(?m)^(Auto-recharge credits|Claude .+?)\s+\d`)
)

// ParseAnthropicInvoice parses an Anthropic dashboard invoice PDF.
func ParseAnthropicInvoice(filePath string) (*Invoice, error) {
	text, err := PdfToText(filePath)
	if err != nil {
		return nil, err
	}

	inv := &Invoice{
		Provider: "anthropic",
		Currency: "USD",
		FilePath: filePath,
	}

	// Invoice number
	if m := reAnthropicInvoiceNum.FindStringSubmatch(text); len(m) == 2 {
		inv.InvoiceNumber = m[1]
	} else {
		return nil, fmt.Errorf("no invoice number found in %s", filePath)
	}

	// Date of issue
	if m := reAnthropicDate.FindStringSubmatch(text); len(m) == 2 {
		if t, err := time.Parse("January 2, 2006", m[1]); err == nil {
			inv.Date = t
		}
	}

	// Amount due (e.g. "$45.36 USD due")
	if m := reAnthropicAmountDue.FindStringSubmatch(text); len(m) == 2 {
		cleaned := strings.ReplaceAll(m[1], ",", "")
		if amt, err := strconv.ParseFloat(cleaned, 64); err == nil {
			inv.Amount = amt
		}
	}

	// Description
	if m := reAnthropicDescription.FindStringSubmatch(text); len(m) == 2 {
		inv.PaymentMethod = "" // Anthropic invoices don't include payment method
	}

	return inv, nil
}
