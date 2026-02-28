package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Scapia date formats in the transaction lines.
// New format: "21-10-2025 · 12:08"
// Old format: "15 Jul 2025 · 16:42"
var (
	// Matches: DD-MM-YYYY · HH:MM MerchantName ... [Payment/Refund] [+]₹Amount [RewardPoints]
	reNewDate = regexp.MustCompile(`^\s*(\d{2}-\d{2}-\d{4})\s*·\s*(\d{2}:\d{2})\s+(.+)$`)
	// Matches: DD Mon YYYY · HH:MM MerchantName ... [Payment/Refund] [+]₹Amount [RewardPoints]
	reOldDate = regexp.MustCompile(`^\s*(\d{1,2}\s+\w{3}\s+\d{4})\s*·\s*(\d{2}:\d{2})\s+(.+)$`)
	// Matches amount: [+]₹X,XXX.XX
	reAmount = regexp.MustCompile(`(\+?)₹([\d,]+\.?\d*)`)
	// Billing cycle period in header
	rePeriod = regexp.MustCompile(`(\d{1,2}\s+\w{3}\s+\d{4})\s*-\s*(\d{1,2}\s+\w{3}\s+\d{4})`)
	// Page break lines to skip
	rePageBreak = regexp.MustCompile(`(?i)^\s*(•?\s*X{10,}|Billing Cycle|\d{1,2}\s+\w{3}\s+\d{4}\s*-\s*\d{1,2}\s+\w{3}\s+\d{4})`)
	// Merchant continuation line (indented text without a date prefix)
	reContinuation = regexp.MustCompile(`^\s{20,}\S`)
)

// ParseScapia parses a Scapia credit card statement PDF.
func ParseScapia(filePath string) (*Statement, error) {
	text, err := PdfToText(filePath)
	if err != nil {
		return nil, err
	}

	stmt := &Statement{
		Provider:    "scapia",
		AccountType: "creditcard",
		FilePath:    filePath,
	}

	// Extract billing period from header
	if m := rePeriod.FindStringSubmatch(text); len(m) == 3 {
		stmt.PeriodFrom, _ = time.Parse("2 Jan 2006", m[1])
		stmt.PeriodTo, _ = time.Parse("2 Jan 2006", m[2])
	}

	// Extract transaction section: between "Your Transactions"/"Your transactions" and "All about your"
	lines := strings.Split(text, "\n")
	inTxnSection := false
	var txnLines []string

	for _, line := range lines {
		lower := strings.ToLower(strings.TrimSpace(line))
		if strings.HasPrefix(lower, "your transactions") {
			inTxnSection = true
			continue
		}
		if inTxnSection && strings.HasPrefix(lower, "all about your") {
			break
		}
		if inTxnSection {
			txnLines = append(txnLines, line)
		}
	}

	// Parse transaction lines
	for _, line := range txnLines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Skip page break lines
		if rePageBreak.MatchString(line) {
			continue
		}

		// Skip continuation lines (wrapped merchant names) - we already captured the main line
		if reContinuation.MatchString(line) && !reNewDate.MatchString(line) && !reOldDate.MatchString(line) {
			continue
		}

		// Try new date format first: DD-MM-YYYY · HH:MM rest
		if m := reNewDate.FindStringSubmatch(line); len(m) == 4 {
			txn, err := parseScapiaTxnLine(m[1], "02-01-2006", m[2], m[3])
			if err == nil {
				stmt.Transactions = append(stmt.Transactions, txn)
			}
			continue
		}

		// Try old date format: DD Mon YYYY · HH:MM rest
		if m := reOldDate.FindStringSubmatch(line); len(m) == 4 {
			txn, err := parseScapiaTxnLine(m[1], "2 Jan 2006", m[2], m[3])
			if err == nil {
				stmt.Transactions = append(stmt.Transactions, txn)
			}
			continue
		}
	}

	return stmt, nil
}

// parseScapiaTxnLine parses the rest of a transaction line after the date/time.
// rest contains: "MerchantName  [Payment/Refund]  [+]₹Amount  [RewardPoints]"
func parseScapiaTxnLine(dateStr, dateLayout, timeStr, rest string) (Transaction, error) {
	txn := Transaction{Currency: "INR"}

	// Parse date
	d, err := time.Parse(dateLayout, dateStr)
	if err != nil {
		return txn, fmt.Errorf("parse date %q: %w", dateStr, err)
	}
	t, err := time.Parse("15:04", timeStr)
	if err != nil {
		return txn, fmt.Errorf("parse time %q: %w", timeStr, err)
	}
	txn.Date = time.Date(d.Year(), d.Month(), d.Day(), t.Hour(), t.Minute(), 0, 0, time.UTC)

	// Determine type and extract amount
	txn.Type = "debit"
	txn.RawLine = strings.TrimSpace(rest)

	// Check for Payment/Refund labels
	lowerRest := strings.ToLower(rest)
	if strings.Contains(lowerRest, "payment") {
		txn.Type = "payment"
	} else if strings.Contains(lowerRest, "refund") {
		txn.Type = "refund"
	}

	// Extract amount
	if m := reAmount.FindStringSubmatch(rest); len(m) == 3 {
		if m[1] == "+" {
			if txn.Type == "debit" {
				txn.Type = "credit" // + prefix without label
			}
		}
		amtStr := strings.ReplaceAll(m[2], ",", "")
		amt, err := strconv.ParseFloat(amtStr, 64)
		if err != nil {
			return txn, fmt.Errorf("parse amount %q: %w", m[2], err)
		}
		txn.Amount = amt
	}

	// Extract merchant name: everything before the amount/type markers
	desc := rest
	// Remove amount portion
	if idx := strings.Index(desc, "₹"); idx > 0 {
		desc = desc[:idx]
	}
	// Remove Payment/Refund labels
	desc = regexp.MustCompile(`(?i)\s+(Payment|Refund)\s*$`).ReplaceAllString(desc, "")
	// Remove trailing +
	desc = strings.TrimRight(desc, "+ ")
	txn.Description = strings.TrimSpace(desc)

	if txn.Description == "" {
		return txn, fmt.Errorf("empty description")
	}

	return txn, nil
}
