package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	reFreepikInvoiceNum    = regexp.MustCompile(`Invoice\s*#\s*(INV-[\w-]+)`)
	reFreepikInvoiceDate   = regexp.MustCompile(`Invoice Date\s+(\w+ \d{1,2}, \d{4})`)
	reFreepikInvoiceAmount = regexp.MustCompile(`Invoice Amount\s+([₹$€])([\d,]+\.?\d*)`)
	reFreepikBillingPeriod = regexp.MustCompile(`Billing Period\s+(\w+ \d{1,2}) to (\w+ \d{1,2}, \d{4})`)
	reFreepikPayment       = regexp.MustCompile(`was paid on .+ by (\w+(?:\s*\w+)?)\s+card ending (\d+)`)
)

// ParseFreepikInvoice parses a Freepik dashboard invoice PDF.
func ParseFreepikInvoice(filePath string) (*Invoice, error) {
	text, err := PdfToText(filePath)
	if err != nil {
		return nil, err
	}

	inv := &Invoice{
		Provider: "freepik",
		FilePath: filePath,
	}

	// Invoice number
	if m := reFreepikInvoiceNum.FindStringSubmatch(text); len(m) == 2 {
		inv.InvoiceNumber = m[1]
	} else {
		return nil, fmt.Errorf("no invoice number found in %s", filePath)
	}

	// Invoice date
	if m := reFreepikInvoiceDate.FindStringSubmatch(text); len(m) == 2 {
		if t, err := time.Parse("Jan 2, 2006", m[1]); err == nil {
			inv.Date = t
		}
	}

	// Invoice amount + currency
	if m := reFreepikInvoiceAmount.FindStringSubmatch(text); len(m) == 3 {
		switch m[1] {
		case "₹":
			inv.Currency = "INR"
		case "$":
			inv.Currency = "USD"
		case "€":
			inv.Currency = "EUR"
		}
		cleaned := strings.ReplaceAll(m[2], ",", "")
		if amt, err := strconv.ParseFloat(cleaned, 64); err == nil {
			inv.Amount = amt
		}
	}

	// Billing period
	if m := reFreepikBillingPeriod.FindStringSubmatch(text); len(m) == 3 {
		// m[1] = "Sep 29", m[2] = "Oct 29, 2025"
		endDate, err := time.Parse("Jan 2, 2006", m[2])
		if err == nil {
			inv.PeriodTo = endDate
			// Start date uses same year (or previous year if month wraps)
			startStr := m[1] + ", " + strconv.Itoa(endDate.Year())
			if t, err := time.Parse("Jan 2, 2006", startStr); err == nil {
				if t.After(endDate) {
					t = t.AddDate(-1, 0, 0)
				}
				inv.PeriodFrom = t
			}
		}
	}

	// Payment method
	if m := reFreepikPayment.FindStringSubmatch(text); len(m) == 3 {
		inv.PaymentMethod = fmt.Sprintf("%s ending %s", m[1], m[2])
	}

	return inv, nil
}
