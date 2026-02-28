package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	// Matches: "09 Jul '25" at the start of a line (full date)
	reAxisCCDate = regexp.MustCompile(`^(\d{2}\s+\w{3}\s+'\d{2})`)
	// Matches: "11 Jan" at the start of a line (partial date, no year)
	reAxisCCDatePartial = regexp.MustCompile(`^(\d{2}\s+\w{3})\s*$`)
	// Matches: "'26" at the start of a line (year suffix)
	reAxisCCYearSuffix = regexp.MustCompile(`^'\d{2}`)
	// Matches: ₹ 13,275.00 followed by Debit/Credit
	reAxisCCAmount = regexp.MustCompile(`₹\s*([\d,]+\.\d{2})\s+(Debit|Credit)`)
	// Matches statement month on a line by itself: " Jul 2025  ..."
	reAxisCCMonth = regexp.MustCompile(`^\s*(\w{3}\s+\d{4})\s`)
)

// ParseAxisCC parses an Axis Bank credit card statement PDF.
func ParseAxisCC(filePath string, password string) (*Statement, error) {
	text, err := PdfToText(filePath, password)
	if err != nil {
		return nil, err
	}

	stmt := &Statement{
		Provider:    "axis_cc",
		AccountType: "creditcard",
		FilePath:    filePath,
	}

	// Extract statement month — appears on the line after "Selected Statement Month"
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.Contains(line, "Selected Statement Month") && i+1 < len(lines) {
			if m := reAxisCCMonth.FindStringSubmatch(lines[i+1]); len(m) == 2 {
				if t, err := time.Parse("Jan 2006", m[1]); err == nil {
					stmt.PeriodFrom = t
					stmt.PeriodTo = t.AddDate(0, 1, -1)
				}
			}
			break
		}
	}

	// Preprocess: normalize split-date lines into single lines
	normalized := normalizeAxisCCLines(text)

	for _, line := range normalized {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		dateMatch := reAxisCCDate.FindStringSubmatch(trimmed)
		amountMatch := reAxisCCAmount.FindStringSubmatch(trimmed)

		if dateMatch == nil || amountMatch == nil {
			continue
		}

		txnDate, err := parseAxisCCDate(dateMatch[1])
		if err != nil {
			continue
		}

		desc := extractAxisCCDesc(trimmed, dateMatch[0])
		txn := buildAxisCCTxn(txnDate, desc, amountMatch, trimmed)
		stmt.Transactions = append(stmt.Transactions, txn)
	}

	if len(stmt.Transactions) == 0 {
		return nil, fmt.Errorf("no transactions found in %s", filePath)
	}

	return stmt, nil
}

// normalizeAxisCCLines preprocesses statement text to handle two formats:
// Format A (normal): "09 Jul '25  description  ₹ amount  Debit/Credit"
// Format B (split):  "11 Jan\n  description  ₹ amount  Debit/Credit\n'26"
// Both get normalized to: "DD Mon 'YY  description  ₹ amount  Debit/Credit"
func normalizeAxisCCLines(text string) []string {
	rawLines := strings.Split(text, "\n")
	var result []string

	inTransactions := false
	i := 0
	for i < len(rawLines) {
		trimmed := strings.TrimSpace(rawLines[i])

		if strings.Contains(trimmed, "Transaction Summary") {
			inTransactions = true
			i++
			continue
		}
		if strings.Contains(trimmed, "End of Transaction Summary") {
			break
		}
		if !inTransactions {
			i++
			continue
		}
		// Skip headers, page breaks, empty
		if trimmed == "" || strings.HasPrefix(trimmed, "Date") || strings.HasPrefix(trimmed, "Page ") ||
			strings.Contains(trimmed, "Amount") && strings.Contains(trimmed, "(INR)") ||
			trimmed == "Debit/Credit" {
			i++
			continue
		}

		// Check if this is a full date line (normal format)
		if reAxisCCDate.MatchString(trimmed) {
			result = append(result, trimmed)
			i++
			continue
		}

		// Check if this is a partial date line (split format: "11 Jan")
		partialMatch := reAxisCCDatePartial.FindStringSubmatch(trimmed)
		if partialMatch != nil {
			// Collect content lines until we find the year suffix
			dateFragment := partialMatch[1]
			var contentParts []string
			i++
			yearSuffix := ""
			for i < len(rawLines) {
				next := strings.TrimSpace(rawLines[i])
				if next == "" {
					i++
					continue
				}
				// Check if this line starts with year suffix like "'26"
				if reAxisCCYearSuffix.MatchString(next) {
					yearSuffix = next[:3] // take "'YY"
					// There might be more description after the year suffix
					extra := strings.TrimSpace(next[3:])
					if extra != "" {
						contentParts = append(contentParts, extra)
					}
					i++
					break
				}
				contentParts = append(contentParts, next)
				i++
			}
			// Reconstruct as: "DD Mon 'YY  content"
			if yearSuffix != "" {
				fullDate := dateFragment + " " + yearSuffix
				content := strings.Join(contentParts, " ")
				merged := fullDate + "  " + content
				result = append(result, merged)
			}
			continue
		}

		// Other line (continuation of previous, skip)
		i++
	}

	return result
}

func buildAxisCCTxn(date time.Time, desc string, amountMatch []string, rawLine string) Transaction {
	amount, _ := strconv.ParseFloat(strings.ReplaceAll(amountMatch[1], ",", ""), 64)
	txnType := "debit"
	if strings.ToLower(amountMatch[2]) == "credit" {
		txnType = "credit"
	}
	return Transaction{
		Date:        date,
		Description: strings.TrimSpace(desc),
		Amount:      amount,
		Currency:    "INR",
		Type:        txnType,
		RawLine:     rawLine,
	}
}

func extractAxisCCDesc(line, dateStr string) string {
	afterDate := strings.TrimSpace(line[len(dateStr):])
	idx := strings.Index(afterDate, "₹")
	if idx > 0 {
		return strings.TrimSpace(afterDate[:idx])
	}
	return afterDate
}

func parseAxisCCDate(s string) (time.Time, error) {
	// "09 Jul '25" → time.Time
	t, err := time.Parse("02 Jan '06", s)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}
