package parser

import "fmt"

// StatementParserFunc parses a financial statement file (PDF/CSV) with an optional password.
type StatementParserFunc func(filePath, password string) (*Statement, error)

// InvoiceParserFunc parses an invoice PDF.
type InvoiceParserFunc func(filePath string) (*Invoice, error)

var (
	emailParsers     = map[string]EmailParserFunc{}
	statementParsers = map[string]StatementParserFunc{}
	invoiceParsers   = map[string]InvoiceParserFunc{}
)

func init() {
	// Email parsers
	emailParsers["anthropic"] = ParseAnthropicEmail
	emailParsers["github"] = ParseGitHubEmail
	emailParsers["godaddy"] = ParseGoDaddyEmail
	emailParsers["freepik"] = ParseFreepikEmail
	emailParsers["wework"] = ParseWeWorkEmail

	// Statement parsers (wrapped to uniform signature)
	statementParsers["scapia"] = func(filePath, _ string) (*Statement, error) {
		return ParseScapia(filePath)
	}
	statementParsers["axis"] = func(filePath, _ string) (*Statement, error) {
		return ParseAxis(filePath)
	}
	statementParsers["axis_cc"] = func(filePath, password string) (*Statement, error) {
		return ParseAxisCC(filePath, password)
	}
	statementParsers["unknown"] = func(filePath, _ string) (*Statement, error) {
		return ParseUnknownSource(filePath)
	}

	// Invoice parsers
	invoiceParsers["freepik"] = ParseFreepikInvoice
	invoiceParsers["anthropic"] = ParseAnthropicInvoice
}

// GetEmailParser returns a registered email parser by name.
func GetEmailParser(name string) (EmailParserFunc, error) {
	p, ok := emailParsers[name]
	if !ok {
		return nil, fmt.Errorf("unknown email parser: %q", name)
	}
	return p, nil
}

// GetStatementParser returns a registered statement parser by name.
func GetStatementParser(name string) (StatementParserFunc, error) {
	p, ok := statementParsers[name]
	if !ok {
		return nil, fmt.Errorf("unknown statement parser: %q", name)
	}
	return p, nil
}

// GetInvoiceParser returns a registered invoice parser by name.
func GetInvoiceParser(name string) (InvoiceParserFunc, error) {
	p, ok := invoiceParsers[name]
	if !ok {
		return nil, fmt.Errorf("unknown invoice parser: %q", name)
	}
	return p, nil
}
