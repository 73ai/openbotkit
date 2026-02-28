package parser

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// ParseUnknownSource parses a CSV file representing transactions from an unknown
// or unavailable payment source (e.g. a closed credit card). The CSV format is:
//
//	Date,Description,Amount,Currency,Type
//	2025-09-10,WeWork myHQ order#395011,1593.00,INR,debit
func ParseUnknownSource(filePath string) (*Statement, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", filePath, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	rows, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read csv %s: %w", filePath, err)
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("no data rows in %s", filePath)
	}

	// Map header columns
	header := rows[0]
	colIdx := map[string]int{}
	for i, col := range header {
		colIdx[strings.TrimSpace(col)] = i
	}

	stmt := &Statement{
		Provider:    "unknown",
		AccountType: "unknown",
		FilePath:    filePath,
	}

	for _, row := range rows[1:] {
		dateStr := strings.TrimSpace(row[colIdx["Date"]])
		desc := strings.TrimSpace(row[colIdx["Description"]])
		amtStr := strings.TrimSpace(row[colIdx["Amount"]])
		currency := strings.TrimSpace(row[colIdx["Currency"]])
		txnType := strings.TrimSpace(row[colIdx["Type"]])

		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return nil, fmt.Errorf("parse date %q: %w", dateStr, err)
		}

		amount, err := strconv.ParseFloat(strings.ReplaceAll(amtStr, ",", ""), 64)
		if err != nil {
			return nil, fmt.Errorf("parse amount %q: %w", amtStr, err)
		}

		stmt.Transactions = append(stmt.Transactions, Transaction{
			Date:        date,
			Description: desc,
			Amount:      amount,
			Currency:    currency,
			Type:        txnType,
			RawLine:     strings.Join(row, ","),
		})

		// Track period range
		if stmt.PeriodFrom.IsZero() || date.Before(stmt.PeriodFrom) {
			stmt.PeriodFrom = date
		}
		if date.After(stmt.PeriodTo) {
			stmt.PeriodTo = date
		}
	}

	return stmt, nil
}
