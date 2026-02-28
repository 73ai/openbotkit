package recon

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/priyanshujain/reimbursement/store"
)

const (
	StatusReconciled   = "RECONCILED"
	StatusUnreconciled = "UNRECONCILED"
)

// ReconResult represents a single reimbursable transaction with reconciliation status.
type ReconResult struct {
	Service        string
	Status         string // RECONCILED or UNRECONCILED
	Date           time.Time
	SourceAmount   string // from source (email/invoice)
	DestAmount     string // from destination (financial statement), or "MANUAL" if unreconciled
	Description    string // email subject or CC description
	Account        string // email account
	AttachmentPath string // path to invoice/receipt PDF
	DestRef        string // destination statement file, if reconciled
	Notes          string // e.g. "Paid via Visa ending 1646"
	AmountINR      string // final INR amount for reimbursement claim
	Source         string // "email", "dashboard"
	Destination    string // e.g. "credit card (scapia)", "bank account (axis)"
	EmailID        int64
}

// reAmount extracts amounts with currency symbols: $118.00, ₹1,200.00, €50.00
var reAmount = regexp.MustCompile(`([$₹€])\s*([\d,]+\.?\d*)`)

// reAmountCode extracts amounts with currency codes: INR 849.6, USD 100.00
var reAmountCode = regexp.MustCompile(`(?i)(INR|USD|EUR)\s+([\d,]+\.?\d*)`)

// reTotalAmount extracts amounts from "Total: $137.18 USD" lines (e.g. GitHub receipts)
var reTotalAmount = regexp.MustCompile(`(?i)Total:\s*([$₹€])\s*([\d,]+\.?\d*)`)

// reChargedTo extracts "Charged to: Visa (4*** **** **** 1646)" style lines
var reChargedTo = regexp.MustCompile(`(?i)charged to:\s*(\w+)\s*\([*\d\s]*?(\d{4})\)`)

// rePaymentMethod extracts "Payment method - 1646" or "Payment method Link" style lines
var rePaymentMethod = regexp.MustCompile(`(?i)payment method\s*[-–]?\s*(\d{4}|Link)`)

// RunRecon performs the full reconciliation:
// 1. For each service, find matching emails (using per-service AfterDate)
// 2. For each email match, try to find a destination statement transaction within ±5 days
// 3. Return all results with RECONCILED/UNRECONCILED status
func RunRecon(services []Service, emailDB *store.Store, stmtDB *store.StatementStore) ([]ReconResult, error) {
	var results []ReconResult
	usedTxns := make(map[int64]bool)

	for _, svc := range services {
		if len(svc.EmailFroms) == 0 && svc.DashboardInvoice == "" && svc.OfflineDir == "" && len(svc.DestPatterns) == 0 {
			continue
		}

		svcResults, err := reconService(svc, emailDB, stmtDB, usedTxns)
		if err != nil {
			return nil, fmt.Errorf("recon %s: %w", svc.Name, err)
		}
		results = append(results, svcResults...)
	}

	return results, nil
}

func reconService(svc Service, emailDB *store.Store, stmtDB *store.StatementStore, usedTxns map[int64]bool) ([]ReconResult, error) {
	var results []ReconResult

	// Source 1: Email invoices
	if len(svc.EmailFroms) > 0 {
		emailResults, err := reconFromEmails(svc, emailDB, stmtDB, usedTxns)
		if err != nil {
			return nil, err
		}
		results = append(results, emailResults...)
	}

	// Source 2: Offline payments (scanned invoices)
	if svc.OfflineDir != "" {
		offlineResults, err := reconFromOffline(svc)
		if err != nil {
			return nil, err
		}
		results = append(results, offlineResults...)
	}

	// Source 3: Dashboard invoices
	if svc.DashboardInvoice != "" {
		dashResults, err := reconFromDashboard(svc, stmtDB, usedTxns)
		if err != nil {
			return nil, err
		}
		// Merge: dashboard invoices enrich or add to email results
		results = mergeDashboardResults(results, dashResults)
	}

	// Source 4: Unmatched statement transactions (no email/invoice, but on the card)
	if len(svc.DestPatterns) > 0 {
		stmtResults, err := reconFromStatements(svc, stmtDB, usedTxns)
		if err != nil {
			return nil, err
		}
		results = append(results, stmtResults...)
	}

	// Post-process: drop unreconciled items beyond statement coverage
	// (we don't have statements downloaded yet for that period).
	if len(svc.DestPatterns) > 0 {
		latest, hasStmts := latestStmtDate(stmtDB, svc.DestPatterns)
		if hasStmts {
			latestDate := time.Date(latest.Year(), latest.Month(), latest.Day(), 0, 0, 0, 0, time.UTC)
			cutoff := latestDate.AddDate(0, 0, 5)
			var filtered []ReconResult
			for i := range results {
				if results[i].Status == StatusUnreconciled && results[i].Date.After(cutoff) {
					continue // beyond statement coverage — skip
				}
				filtered = append(filtered, results[i])
			}
			results = filtered
		}
	}

	return results, nil
}

func reconFromEmails(svc Service, emailDB *store.Store, stmtDB *store.StatementStore, usedTxns map[int64]bool) ([]ReconResult, error) {
	emails, err := emailDB.SearchEmails(svc.EmailFroms, svc.AfterDate)
	if err != nil {
		return nil, fmt.Errorf("search emails: %w", err)
	}

	// Pre-fetch dashboard invoice numbers for deduplication — if an email
	// references a dashboard invoice, skip it (the dashboard version matches the statement).
	var dashInvoiceNums map[string]bool
	if svc.DashboardInvoice != "" {
		invoices, _ := stmtDB.SearchInvoices(svc.DashboardInvoice)
		if len(invoices) > 0 {
			dashInvoiceNums = make(map[string]bool, len(invoices))
			for _, inv := range invoices {
				dashInvoiceNums[inv.InvoiceNumber] = true
			}
		}
	}

	// Build all email results first (without statement matching).
	type pendingEmail struct {
		result  ReconResult
		ccMatch *store.StatementTxn
	}
	var pending []pendingEmail

	for _, email := range emails {
		if !matchesSubject(email.Subject, svc.EmailSubjects) {
			continue
		}
		if shouldExclude(email.Subject, svc.ExcludeSubject) {
			continue
		}

		// Skip emails that reference a known dashboard invoice number
		if len(dashInvoiceNums) > 0 {
			skip := false
			for invNum := range dashInvoiceNums {
				if strings.Contains(email.Body, invNum) || strings.Contains(email.HTMLBody, invNum) {
					skip = true
					break
				}
			}
			if skip {
				continue
			}
		}

		result := ReconResult{
			Service:     svc.Name,
			Status:      StatusUnreconciled,
			Date:        email.Date,
			DestAmount:  "MANUAL",
			Description: email.Subject,
			Account:     email.Account,
			Source:      "email",
			EmailID:     email.ID,
		}

		// Per-provider parser when available, generic fallback otherwise
		if svc.EmailParser != nil {
			parsed := svc.EmailParser(email.Subject, email.Body, email.HTMLBody)
			result.SourceAmount = parsed.Amount
			result.Notes = parsed.PaymentMethod
		} else {
			searchText := email.Body + " " + email.Subject
			result.SourceAmount = extractAmount(searchText)
			if result.SourceAmount == "" {
				result.SourceAmount = extractAmount(email.HTMLBody)
			}
			result.Notes = extractPaymentMethod(email.Body)
			if result.Notes == "" {
				result.Notes = extractPaymentMethod(email.HTMLBody)
			}
		}

		if email.AttachmentPath != "" {
			result.AttachmentPath = email.AttachmentPath
		}

		pending = append(pending, pendingEmail{result: result})
	}

	// Two-pass statement matching to prevent chain displacement.
	// Pass 1 (tight): match only within ±1 day — locks in same-day matches first.
	// Pass 2 (loose): match remaining within ±5 days.
	if len(svc.DestPatterns) > 0 {
		for i := range pending {
			ccMatch, err := findDestMatch(stmtDB, svc.DestPatterns, pending[i].result.Date, usedTxns, pending[i].result.SourceAmount, 1)
			if err == nil && ccMatch != nil {
				pending[i].result.Status = StatusReconciled
				pending[i].result.DestAmount = fmt.Sprintf("%.2f", ccMatch.Amount)
				pending[i].result.DestRef = ccMatch.FilePath
				pending[i].result.Destination = formatDestination(ccMatch.AccountType, ccMatch.Provider)
				pending[i].ccMatch = ccMatch
			}
		}

		for i := range pending {
			if pending[i].result.Status != StatusUnreconciled {
				continue
			}
			ccMatch, err := findDestMatch(stmtDB, svc.DestPatterns, pending[i].result.Date, usedTxns, pending[i].result.SourceAmount, 5)
			if err == nil && ccMatch != nil {
				pending[i].result.Status = StatusReconciled
				pending[i].result.DestAmount = fmt.Sprintf("%.2f", ccMatch.Amount)
				pending[i].result.DestRef = ccMatch.FilePath
				pending[i].result.Destination = formatDestination(ccMatch.AccountType, ccMatch.Provider)
				pending[i].ccMatch = ccMatch
			}
		}
	}

	// Collect results and add surcharges for reconciled items.
	var results []ReconResult
	for i := range pending {
		if pending[i].ccMatch != nil && len(svc.Surcharges) > 0 {
			surcharges := findDestSurcharges(stmtDB, svc.Name, svc.Surcharges, pending[i].ccMatch, pending[i].result.Description, pending[i].result.Account, pending[i].result.Destination, usedTxns)
			results = append(results, surcharges...)
		}
		resolveINR(&pending[i].result)
		results = append(results, pending[i].result)
	}

	return results, nil
}

func reconFromDashboard(svc Service, stmtDB *store.StatementStore, usedTxns map[int64]bool) ([]ReconResult, error) {
	invoices, err := stmtDB.SearchInvoices(svc.DashboardInvoice)
	if err != nil {
		return nil, fmt.Errorf("search dashboard invoices: %w", err)
	}

	var results []ReconResult

	for _, inv := range invoices {
		// Exclude the AfterDate itself (already reimbursed)
		afterNextDay := svc.AfterDate.AddDate(0, 0, 1)
		if inv.Date.Before(afterNextDay) {
			continue
		}

		// Format source amount with currency symbol
		symbol := currencySymbol(inv.Currency)
		sourceAmt := fmt.Sprintf("%s%.2f", symbol, inv.Amount)

		result := ReconResult{
			Service:        svc.Name,
			Status:         StatusUnreconciled,
			Date:           inv.Date,
			SourceAmount:   sourceAmt,
			DestAmount:       "MANUAL",
			Description:    fmt.Sprintf("%s invoice %s", svc.Name, inv.InvoiceNumber),
			AttachmentPath: inv.FilePath,
			Source:         "dashboard",
		}

		if inv.PaymentMethod != "" {
			result.Notes = "Paid via " + inv.PaymentMethod
		}

		// Try dest match
		if len(svc.DestPatterns) > 0 {
			ccMatch, err := findDestMatch(stmtDB, svc.DestPatterns, inv.Date, usedTxns, sourceAmt)
			if err == nil && ccMatch != nil {
				result.Status = StatusReconciled
				result.DestAmount = fmt.Sprintf("%.2f", ccMatch.Amount)
				result.DestRef = ccMatch.FilePath
				result.Destination = formatDestination(ccMatch.AccountType, ccMatch.Provider)
			}
		}

		resolveINR(&result)
		results = append(results, result)
	}

	return results, nil
}

// reconFromOffline reads a manifest.csv from the offline directory.
// Each row represents an offline payment with a scanned invoice PDF.
// These are auto-reconciled — paper invoices paid in person, no payment proof needed.
// manifest.csv format: Date,Amount,Description,Invoice
func reconFromOffline(svc Service) ([]ReconResult, error) {
	manifestPath := filepath.Join(svc.OfflineDir, "manifest.csv")

	f, err := os.Open(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("open offline manifest %s: %w", manifestPath, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	rows, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read offline manifest: %w", err)
	}

	if len(rows) < 2 {
		return nil, nil
	}

	// Map header columns
	header := rows[0]
	colIdx := map[string]int{}
	for i, col := range header {
		colIdx[strings.TrimSpace(col)] = i
	}

	dateCol := colIdx["Date"]
	amountCol := colIdx["Amount"]
	descCol, hasDesc := colIdx["Description"]
	invoiceCol, hasInvoice := colIdx["Invoice"]

	var results []ReconResult

	for _, row := range rows[1:] {
		if len(row) <= dateCol || len(row) <= amountCol {
			continue
		}

		dateStr := strings.TrimSpace(row[dateCol])
		amount := strings.TrimSpace(row[amountCol])

		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		// Exclude the AfterDate itself (already reimbursed)
		if !svc.AfterDate.IsZero() && date.Before(svc.AfterDate.AddDate(0, 0, 1)) {
			continue
		}

		desc := svc.Name + " offline payment"
		if hasDesc && descCol < len(row) && strings.TrimSpace(row[descCol]) != "" {
			desc = strings.TrimSpace(row[descCol])
		}

		var attachPath string
		if hasInvoice && invoiceCol < len(row) && strings.TrimSpace(row[invoiceCol]) != "" {
			attachPath = filepath.Join(svc.OfflineDir, strings.TrimSpace(row[invoiceCol]))
		}

		result := ReconResult{
			Service:        svc.Name,
			Status:         StatusReconciled,
			Date:           date,
			SourceAmount:   "₹" + amount,
			DestAmount:     amount,
			Description:    desc,
			AttachmentPath: attachPath,
			Source:         "offline",
			Notes:          "Offline payment (paper invoice)",
			AmountINR:      "₹" + amount,
		}

		results = append(results, result)
	}

	return results, nil
}

// reconFromStatements picks up statement transactions that match the service's DestPatterns
// but were not matched to any email or dashboard invoice. These are charges confirmed on
// the credit card/bank but with no corresponding email receipt (e.g., Anthropic API usage).
func reconFromStatements(svc Service, stmtDB *store.StatementStore, usedTxns map[int64]bool) ([]ReconResult, error) {
	var results []ReconResult

	for _, pattern := range svc.DestPatterns {
		txns, err := stmtDB.SearchTransactions(pattern)
		if err != nil {
			continue
		}

		for _, txn := range txns {
			if txn.TxnType != "debit" {
				continue
			}
			if usedTxns[txn.ID] {
				continue // already matched to an email/invoice
			}
			// Skip transactions on or before AfterDate (exclusive: the full day is excluded)
			if !svc.AfterDate.IsZero() && txn.Date.Before(svc.AfterDate.AddDate(0, 0, 1)) {
				continue
			}

			usedTxns[txn.ID] = true

			result := ReconResult{
				Service:     svc.Name,
				Status:      StatusReconciled,
				Date:        txn.Date,
				SourceAmount: fmt.Sprintf("₹%.2f", txn.Amount),
				DestAmount:  fmt.Sprintf("%.2f", txn.Amount),
				Description: txn.Description,
				Source:      "statement",
				Destination: formatDestination(txn.AccountType, txn.Provider),
				DestRef:     txn.FilePath,
				Notes:       "Statement only (no email receipt)",
				AmountINR:   fmt.Sprintf("₹%.2f", txn.Amount),
			}
			results = append(results, result)
		}
	}

	return results, nil
}

// mergeDashboardResults merges dashboard results into email results.
// If a dashboard invoice matches an email result by date (±2 days), the dashboard
// data enriches the email result (adds source amount, better attachment).
// Dashboard invoices with no email match are added as new results.
func mergeDashboardResults(emailResults, dashResults []ReconResult) []ReconResult {
	used := make(map[int]bool) // track which dashboard results were merged

	for i := range emailResults {
		for j := range dashResults {
			if used[j] {
				continue
			}
			diff := math.Abs(emailResults[i].Date.Sub(dashResults[j].Date).Hours())
			if diff <= 2*24 && amountsCompatible(emailResults[i].SourceAmount, dashResults[j].SourceAmount) {
				// Dashboard enriches email result
				if emailResults[i].SourceAmount == "" && dashResults[j].SourceAmount != "" {
					emailResults[i].SourceAmount = dashResults[j].SourceAmount
				}
				if emailResults[i].Notes == "" && dashResults[j].Notes != "" {
					emailResults[i].Notes = dashResults[j].Notes
				}
				if dashResults[j].DestAmount != "MANUAL" && emailResults[i].DestAmount == "MANUAL" {
					emailResults[i].Status = dashResults[j].Status
					emailResults[i].DestAmount = dashResults[j].DestAmount
					emailResults[i].DestRef = dashResults[j].DestRef
					emailResults[i].Destination = dashResults[j].Destination
				}
				resolveINR(&emailResults[i]) // re-resolve after enrichment
				used[j] = true
				break
			}
		}
	}

	// Add unmatched dashboard invoices as new results
	for j := range dashResults {
		if !used[j] {
			emailResults = append(emailResults, dashResults[j])
		}
	}

	return emailResults
}

// resolveINR determines the final INR amount for reimbursement.
// Priority: dest amount (from statement) > source amount if already INR > MANUAL.
func resolveINR(result *ReconResult) {
	if result.DestAmount != "" && result.DestAmount != "MANUAL" {
		result.AmountINR = "₹" + result.DestAmount
		return
	}
	if strings.HasPrefix(result.SourceAmount, "₹") {
		result.AmountINR = result.SourceAmount
		return
	}
	result.AmountINR = "MANUAL"
}

func currencySymbol(currency string) string {
	switch strings.ToUpper(currency) {
	case "INR":
		return "₹"
	case "USD":
		return "$"
	case "EUR":
		return "€"
	default:
		return currency + " "
	}
}

// formatDestination returns a human-readable destination label like "credit card (scapia)" or "bank account (axis)".
func formatDestination(accountType, provider string) string {
	label := accountType
	switch accountType {
	case "creditcard":
		label = "credit card"
	case "bank":
		label = "bank account"
	}
	return fmt.Sprintf("%s (%s)", label, provider)
}

// findDestMatch searches for a destination statement transaction matching any of the patterns
// within ±5 days of the source date. It searches ALL patterns and returns the single best
// match across all of them. When a source amount is available, candidates whose INR amount
// is within 50% of the expected conversion are strongly preferred — if any such candidate
// exists, amount-mismatched candidates are excluded entirely. This prevents e.g. a $118
// subscription email from matching a ₹4,000 API charge when a ₹10,000 subscription charge
// exists on the same day. The usedTxns map enforces 1:1 matching.
func findDestMatch(stmtDB *store.StatementStore, patterns []string, emailDate time.Time, usedTxns map[int64]bool, sourceAmount string, maxDays ...float64) (*store.StatementTxn, error) {
	maxHours := float64(5 * 24) // default ±5 days
	if len(maxDays) > 0 {
		maxHours = maxDays[0] * 24
	}

	expectedINR := estimateINR(sourceAmount)

	type candidate struct {
		txn      *store.StatementTxn
		dateDiff float64
		amtOK    bool // within 50% of expected INR
	}

	var candidates []candidate

	for _, pattern := range patterns {
		txns, err := stmtDB.SearchTransactions(pattern)
		if err != nil {
			continue
		}

		for i := range txns {
			if txns[i].TxnType != "debit" || usedTxns[txns[i].ID] {
				continue
			}
			diff := math.Abs(txns[i].Date.Sub(emailDate).Hours())
			if diff > maxHours {
				continue
			}

			amtOK := false
			if expectedINR > 0 {
				deviation := math.Abs(txns[i].Amount-expectedINR) / expectedINR
				amtOK = deviation <= 0.50
			}

			candidates = append(candidates, candidate{
				txn:      &txns[i],
				dateDiff: diff,
				amtOK:    amtOK,
			})
		}
	}

	if len(candidates) == 0 {
		return nil, sql.ErrNoRows
	}

	// When we know the expected INR amount, only consider amount-compatible candidates.
	// If no candidates pass the amount check, return no match — it's better to leave the
	// item UNRECONCILED than to match it to a wrong-magnitude transaction.
	if expectedINR > 0 {
		var filtered []candidate
		for _, c := range candidates {
			if c.amtOK {
				filtered = append(filtered, c)
			}
		}
		if len(filtered) > 0 {
			candidates = filtered
		} else {
			return nil, sql.ErrNoRows
		}
	}

	var bestMatch *store.StatementTxn
	bestDiff := math.MaxFloat64
	bestIsUnknown := false

	for _, c := range candidates {
		isUnknown := c.txn.AccountType == "unknown"
		better := c.dateDiff < bestDiff
		// For equal date diffs, prefer known providers over "unknown"
		if !better && c.dateDiff == bestDiff && bestIsUnknown && !isUnknown {
			better = true
		}
		if better {
			bestDiff = c.dateDiff
			bestMatch = c.txn
			bestIsUnknown = isUnknown
		}
	}

	if bestMatch != nil {
		usedTxns[bestMatch.ID] = true
		return bestMatch, nil
	}

	return nil, sql.ErrNoRows
}

// estimateINR converts a source amount string (e.g. "$118.00", "₹1,200") to an
// approximate INR value for matching purposes. Uses rough exchange rates — these are
// only used to distinguish obviously different charges, not for final amounts.
func estimateINR(sourceAmt string) float64 {
	if sourceAmt == "" {
		return 0
	}
	type rate struct {
		prefix string
		factor float64
	}
	rates := []rate{
		{"₹", 1.0},
		{"$", 85.0},
		{"€", 93.0},
	}
	for _, r := range rates {
		if strings.HasPrefix(sourceAmt, r.prefix) {
			numStr := strings.TrimPrefix(sourceAmt, r.prefix)
			numStr = strings.ReplaceAll(numStr, ",", "")
			if val, err := strconv.ParseFloat(numStr, 64); err == nil {
				return val * r.factor
			}
		}
	}
	return 0
}

// amountsCompatible returns true if two source amount strings are in the same ballpark
// (within 50% of each other when converted to INR). If either amount is empty, they're
// considered compatible (allows enrichment when one side has no amount).
func amountsCompatible(a, b string) bool {
	inrA := estimateINR(a)
	inrB := estimateINR(b)
	if inrA == 0 || inrB == 0 {
		return true // unknown amount, allow merge
	}
	ratio := inrA / inrB
	return ratio >= 0.5 && ratio <= 2.0
}

// latestStmtDate finds the latest debit transaction date across all matching patterns.
// Used to determine statement coverage — transactions beyond this date + buffer
// likely fall outside our statement data.
func latestStmtDate(stmtDB *store.StatementStore, patterns []string) (time.Time, bool) {
	var latest time.Time
	found := false
	for _, pattern := range patterns {
		txns, err := stmtDB.SearchTransactions(pattern)
		if err != nil {
			continue
		}
		for _, txn := range txns {
			if txn.TxnType == "debit" && txn.Date.After(latest) {
				latest = txn.Date
				found = true
			}
		}
	}
	return latest, found
}

// matchesSubject returns true if the subject contains ALL of the required patterns.
// If patterns is empty, any subject matches.
func matchesSubject(subject string, patterns []string) bool {
	if len(patterns) == 0 {
		return true
	}
	lower := strings.ToLower(subject)
	for _, p := range patterns {
		if !strings.Contains(lower, strings.ToLower(p)) {
			return false
		}
	}
	return true
}

// shouldExclude returns true if the subject contains any of the exclusion patterns.
func shouldExclude(subject string, patterns []string) bool {
	lower := strings.ToLower(subject)
	for _, p := range patterns {
		if strings.Contains(lower, strings.ToLower(p)) {
			return true
		}
	}
	return false
}

// extractAmount finds the best currency amount in a string.
// Prefers "Total: $X.XX" lines (e.g. GitHub receipts with multiple line items)
// over the first occurrence, to avoid grabbing a line-item amount instead of the total.
// Supports: "$118.00", "₹1,200.00", "€50", "INR 849.6", "USD 100.00"
func extractAmount(text string) string {
	// Prefer explicit "Total:" line when present
	if m := reTotalAmount.FindStringSubmatch(text); len(m) == 3 {
		return m[1] + strings.ReplaceAll(m[2], ",", "")
	}
	// Try symbol format: $118.00, ₹1,200.00
	if m := reAmount.FindStringSubmatch(text); len(m) == 3 {
		return m[1] + strings.ReplaceAll(m[2], ",", "")
	}
	// Try code format: INR 849.6, USD 100.00
	if m := reAmountCode.FindStringSubmatch(text); len(m) == 3 {
		symbol := currencySymbol(strings.ToUpper(m[1]))
		return symbol + strings.ReplaceAll(m[2], ",", "")
	}
	return ""
}

// findDestSurcharges looks for bank surcharges on a matched destination transaction
// using the service's surcharge rules. For each rule, it searches for a statement
// transaction matching the pattern with the expected percentage of the parent amount,
// within MaxDays after the parent. If GSTRate is set, it also searches for a GST
// charge on the surcharge amount.
func findDestSurcharges(stmtDB *store.StatementStore, svcName string, rules []SurchargeRule, destMatch *store.StatementTxn, parentDesc, account, destination string, usedTxns map[int64]bool) []ReconResult {
	var surcharges []ReconResult

	for _, rule := range rules {
		expectedFee := math.Round(destMatch.Amount*rule.Percentage*100) / 100

		feeTxns, err := stmtDB.SearchTransactions(rule.Pattern)
		if err != nil {
			continue
		}

		for _, ft := range feeTxns {
			if ft.TxnType != "debit" || usedTxns[ft.ID] {
				continue
			}
			daysDiff := ft.Date.Sub(destMatch.Date).Hours() / 24
			if daysDiff < 0 || daysDiff > float64(rule.MaxDays) {
				continue
			}
			if math.Abs(ft.Amount-expectedFee) > 0.02 {
				continue
			}

			usedTxns[ft.ID] = true
			pctLabel := fmt.Sprintf("%.0f%%", rule.Percentage*100)
			feeResult := ReconResult{
				Service:     svcName,
				Status:      StatusReconciled,
				Date:        ft.Date,
				DestAmount:  fmt.Sprintf("%.2f", ft.Amount),
				Description: fmt.Sprintf("%s (%s)", rule.Pattern, parentDesc),
				Account:     account,
				Source:      "statement surcharge",
				Destination: destination,
				DestRef:     ft.FilePath,
				Notes:       fmt.Sprintf("%s fee on ₹%.2f", pctLabel, destMatch.Amount),
				AmountINR:   fmt.Sprintf("₹%.2f", ft.Amount),
			}
			surcharges = append(surcharges, feeResult)

			// Look for GST on this fee if configured
			if rule.GSTRate > 0 {
				expectedGST := math.Round(ft.Amount*rule.GSTRate*100) / 100
				gstTxns, err := stmtDB.SearchTransactions("GST")
				if err != nil {
					break
				}

				for _, gt := range gstTxns {
					if gt.TxnType != "debit" || usedTxns[gt.ID] {
						continue
					}
					gstDaysDiff := math.Abs(gt.Date.Sub(ft.Date).Hours() / 24)
					if gstDaysDiff > 2 {
						continue
					}
					if math.Abs(gt.Amount-expectedGST) > 0.02 {
						continue
					}

					usedTxns[gt.ID] = true
					gstPctLabel := fmt.Sprintf("%.0f%%", rule.GSTRate*100)
					gstResult := ReconResult{
						Service:     svcName,
						Status:      StatusReconciled,
						Date:        gt.Date,
						DestAmount:  fmt.Sprintf("%.2f", gt.Amount),
						Description: fmt.Sprintf("GST on %s (%s)", rule.Pattern, parentDesc),
						Account:     account,
						Source:      "statement surcharge",
						Destination: destination,
						DestRef:     gt.FilePath,
						Notes:       fmt.Sprintf("%s GST on fee ₹%.2f", gstPctLabel, ft.Amount),
						AmountINR:   fmt.Sprintf("₹%.2f", gt.Amount),
					}
					surcharges = append(surcharges, gstResult)
					break
				}
			}

			break // only one fee match per rule per parent
		}
	}

	return surcharges
}

// extractPaymentMethod finds payment card info from email body.
// Supports: "Charged to: Visa (... 1646)" and "Payment method - 1646" / "Payment method Link".
func extractPaymentMethod(body string) string {
	if m := reChargedTo.FindStringSubmatch(body); len(m) == 3 {
		return fmt.Sprintf("Paid via %s ending %s", m[1], m[2])
	}
	if m := rePaymentMethod.FindStringSubmatch(body); len(m) == 2 {
		if m[1] == "Link" {
			return "Paid via Stripe Link"
		}
		return fmt.Sprintf("Paid via card ending %s", m[1])
	}
	return ""
}
