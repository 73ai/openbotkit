package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	// Matches a date at the start of a line: DD-MM-YYYY
	reAxisDate = regexp.MustCompile(`^(\d{2}-\d{2}-\d{4})\s`)
	// Matches the period header
	reAxisPeriod = regexp.MustCompile(`for the period \(From : (\d{2}-\d{2}-\d{4}) To : (\d{2}-\d{2}-\d{4})\)`)
	// Matches amount fields: numbers with optional commas and decimals
	reAxisAmount = regexp.MustCompile(`([\d,]+\.\d{2})`)
)

// ParseAxis parses an Axis bank account statement PDF.
func ParseAxis(filePath string) (*Statement, error) {
	text, err := PdfToText(filePath)
	if err != nil {
		return nil, err
	}

	stmt := &Statement{
		Provider:    "axis",
		AccountType: "bank",
		FilePath:    filePath,
	}

	// Extract period from header
	if m := reAxisPeriod.FindStringSubmatch(text); len(m) == 3 {
		stmt.PeriodFrom, _ = time.Parse("02-01-2006", m[1])
		stmt.PeriodTo, _ = time.Parse("02-01-2006", m[2])
	}

	lines := strings.Split(text, "\n")

	// Find the transaction table start (after "Tran Date" header)
	startIdx := -1
	for i, line := range lines {
		if strings.Contains(line, "Tran Date") && strings.Contains(line, "Particulars") {
			startIdx = i + 1
			break
		}
	}
	if startIdx == -1 {
		return nil, fmt.Errorf("could not find transaction table header")
	}

	// Skip the separator line (if any blank or header continuation lines)
	for startIdx < len(lines) {
		trimmed := strings.TrimSpace(lines[startIdx])
		if trimmed == "" || trimmed == "Br" {
			startIdx++
			continue
		}
		break
	}

	// Parse transactions. Each entry may span multiple lines:
	// - Continuation lines have no date in the first column
	// - The line with the date has the amounts (debit/credit/balance)
	// - Particulars accumulate across continuation lines + the date line
	var (
		currentParticulars []string
		txnLines           []string
	)

	for i := startIdx; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Stop at end markers
		if strings.HasPrefix(trimmed, "TRANSACTION TOTAL") ||
			strings.HasPrefix(trimmed, "CLOSING BALANCE") ||
			strings.HasPrefix(trimmed, "Unless the constituent") ||
			strings.HasPrefix(trimmed, "++++") {
			break
		}

		if trimmed == "" {
			continue
		}

		txnLines = append(txnLines, line)
	}

	// Process the lines: group by transactions
	// A date line marks the "end" of a transaction entry (amounts are on the date line)
	// Lines without a date are continuation of particulars
	for i := 0; i < len(txnLines); i++ {
		line := txnLines[i]

		if reAxisDate.MatchString(line) {
			// This line has a date - it's the final line of a transaction
			dateStr := reAxisDate.FindStringSubmatch(line)[1]
			// Extract particulars from this line (after the date, before amounts)
			lineParticulars := extractAxisParticulars(line)
			currentParticulars = append(currentParticulars, lineParticulars)

			// Parse the full transaction
			fullParticulars := strings.Join(currentParticulars, " ")
			fullParticulars = strings.TrimSpace(fullParticulars)

			if fullParticulars != "" && fullParticulars != "OPENING BALANCE" {
				txn, err := parseAxisTransaction(dateStr, fullParticulars, line)
				if err == nil {
					stmt.Transactions = append(stmt.Transactions, txn)
				}
			}

			currentParticulars = nil
		} else {
			// Continuation line - accumulate particulars
			part := extractAxisParticulars(line)
			if part != "" {
				currentParticulars = append(currentParticulars, part)
			}
		}
	}

	return stmt, nil
}

// extractAxisParticulars extracts the particulars text from a line.
// The particulars column is roughly between positions 24 and 55.
func extractAxisParticulars(line string) string {
	// For lines with a date, particulars start after the date+spaces
	// For continuation lines, particulars start after the initial spaces
	// We extract text from approximately column 24 to just before the amounts

	if len(line) < 24 {
		return strings.TrimSpace(line)
	}

	// Find where amount columns start by looking for the pattern
	// Amounts are right-aligned and appear after column ~55
	endCol := len(line)

	// Find rightmost portion that looks like amounts
	// Look for the amount pattern starting from around column 50
	rest := ""
	if len(line) > 55 {
		rest = line[55:]
	}

	if rest != "" {
		// Find first amount in the rest
		if idx := reAxisAmount.FindStringIndex(rest); idx != nil {
			endCol = 55 + idx[0]
		}
	}

	startCol := 24
	if endCol <= startCol {
		endCol = len(line)
	}

	if startCol > len(line) {
		return ""
	}
	if endCol > len(line) {
		endCol = len(line)
	}

	return strings.TrimSpace(line[startCol:endCol])
}

// parseAxisTransaction parses a single Axis bank transaction.
func parseAxisTransaction(dateStr, particulars, rawLine string) (Transaction, error) {
	txn := Transaction{
		Currency:    "INR",
		Description: particulars,
		RawLine:     strings.TrimSpace(rawLine),
	}

	d, err := time.Parse("02-01-2006", dateStr)
	if err != nil {
		return txn, fmt.Errorf("parse date %q: %w", dateStr, err)
	}
	txn.Date = d

	// Extract amounts from the line. The layout has columns:
	// ~col 55-68: Debit, ~col 68-82: Credit, ~col 82-100: Balance
	// We look for amounts in the right portion of the line
	if len(rawLine) > 55 {
		amountSection := rawLine[55:]
		amounts := reAxisAmount.FindAllStringSubmatch(amountSection, -1)

		switch len(amounts) {
		case 3:
			// Debit, Credit, Balance - but one of debit/credit is empty
			// Need to figure out which column each amount falls in
			debit, credit := parseAxisAmountColumns(rawLine)
			if debit > 0 {
				txn.Amount = debit
				txn.Type = "debit"
			} else if credit > 0 {
				txn.Amount = credit
				txn.Type = "credit"
			}
		case 2:
			// Either (Debit, Balance) or (Credit, Balance)
			debit, credit := parseAxisAmountColumns(rawLine)
			if debit > 0 {
				txn.Amount = debit
				txn.Type = "debit"
			} else if credit > 0 {
				txn.Amount = credit
				txn.Type = "credit"
			}
		case 1:
			// Just balance (opening balance line) - skip
			return txn, fmt.Errorf("only balance found")
		}
	}

	if txn.Amount == 0 {
		return txn, fmt.Errorf("no amount found")
	}

	return txn, nil
}

// parseAxisAmountColumns uses positional parsing to determine debit vs credit.
// In the layout: Debit is roughly cols 55-68, Credit is roughly cols 68-85, Balance is 85+
func parseAxisAmountColumns(line string) (debit, credit float64) {
	if len(line) < 70 {
		return 0, 0
	}

	// Debit column: approximately position 55-68
	debitSection := ""
	if len(line) > 68 {
		debitSection = line[55:68]
	} else if len(line) > 55 {
		debitSection = line[55:]
	}

	// Credit column: approximately position 68-85
	creditSection := ""
	if len(line) > 85 {
		creditSection = line[68:85]
	} else if len(line) > 68 {
		creditSection = line[68:]
	}

	if m := reAxisAmount.FindStringSubmatch(debitSection); len(m) > 0 {
		amt, _ := strconv.ParseFloat(strings.ReplaceAll(m[1], ",", ""), 64)
		debit = amt
	}

	if m := reAxisAmount.FindStringSubmatch(creditSection); len(m) > 0 {
		amt, _ := strconv.ParseFloat(strings.ReplaceAll(m[1], ",", ""), 64)
		credit = amt
	}

	return debit, credit
}
