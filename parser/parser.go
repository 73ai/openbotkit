package parser

import (
	"fmt"
	"os/exec"
	"time"
)

// Transaction represents a single parsed transaction from a financial statement.
type Transaction struct {
	Date        time.Time
	Description string  // merchant name as-is from statement
	Amount      float64 // always positive
	Currency    string  // "INR"
	Type        string  // "debit", "credit", "refund", "payment"
	RawLine     string  // original text for debugging
}

// Statement represents a parsed financial statement.
type Statement struct {
	Provider     string // "scapia", "axis"
	AccountType  string // "creditcard", "bank"
	PeriodFrom   time.Time
	PeriodTo     time.Time
	FilePath     string
	Transactions []Transaction
}

// Invoice represents a parsed invoice from a service dashboard.
type Invoice struct {
	Provider      string    // "freepik", etc.
	InvoiceNumber string    // e.g. "INV-C-2025-11895700"
	Date          time.Time // invoice date
	Amount        float64   // total amount paid
	Currency      string    // "INR", "USD", "EUR"
	PeriodFrom    time.Time // billing period start
	PeriodTo      time.Time // billing period end
	PaymentMethod string    // e.g. "MasterCard ending 8684"
	FilePath      string    // path to source PDF
}

// ParsedEmail holds the amount and payment method extracted from a receipt email.
type ParsedEmail struct {
	Amount        string // e.g. "$118.00", "₹960.00"
	PaymentMethod string // e.g. "Paid via card ending 1646"
}

// EmailParserFunc is a per-provider email parser.
type EmailParserFunc func(subject, body, htmlBody string) ParsedEmail

// PdfToText extracts text from a PDF file using pdftotext with layout preservation.
// Optional password for encrypted PDFs.
func PdfToText(filePath string, password ...string) (string, error) {
	args := []string{"-layout"}
	if len(password) > 0 && password[0] != "" {
		args = append(args, "-upw", password[0])
	}
	args = append(args, filePath, "-")
	cmd := exec.Command("pdftotext", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("pdftotext %s: %w", filePath, err)
	}
	return string(out), nil
}
