package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/priyanshujain/reimbursement/config"
	"github.com/priyanshujain/reimbursement/gmail"
	"github.com/priyanshujain/reimbursement/parser"
	"github.com/priyanshujain/reimbursement/recon"
	"github.com/priyanshujain/reimbursement/search"
	"github.com/priyanshujain/reimbursement/store"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: reimbursement <command>")
		fmt.Println("Commands:")
		fmt.Println("  fetch   - Fetch emails from Gmail")
		fmt.Println("  parse   - Parse financial statement PDFs into statements.db")
		fmt.Println("  recon   - Reconcile emails with statements and generate reconciled.csv")
		fmt.Println("  package - Package reconciled.csv into a reimbursement folder with attachments and zip")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "fetch":
		runFetch()
	case "parse":
		runParse()
	case "recon":
		runRecon()
	case "package":
		runPackage()
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

// --- fetch command ---

func runFetch() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	if len(cfg.Accounts) == 0 {
		log.Fatal("No accounts configured in config.yaml")
	}

	services, err := config.BuildServices(cfg)
	if err != nil {
		log.Fatalf("Failed to build services: %v", err)
	}

	db, err := store.New("data/emails.db")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	client, err := gmail.NewClient("credentials.json", "data/credentials.db", cfg.Accounts)
	if err != nil {
		log.Fatalf("Failed to create Gmail client: %v", err)
	}

	sources := config.FetchSourcesFromServices(services)
	limiter := search.NewRateLimiter()

	for _, account := range client.Accounts {
		for _, from := range sources {
			query := search.FetchQuery{From: from, After: cfg.AfterDate}
			fmt.Printf("Fetching from %s in %s...\n", from, account.Email)

			msgIDs, err := search.SearchIDs(account.Service, query, limiter)
			if err != nil {
				log.Printf("Error searching %s: %v", from, err)
				continue
			}

			fmt.Printf("  Found %d messages\n", len(msgIDs))

			fetched := 0
			skipped := 0
			for _, id := range msgIDs {
				exists, err := db.EmailExists(id, account.Email)
				if err != nil {
					log.Printf("Error checking email %s: %v", id, err)
					continue
				}
				if exists {
					skipped++
					continue
				}

				email, err := search.FetchEmail(account.Service, account.Email, id, limiter)
				if err != nil {
					log.Printf("Error fetching email %s: %v", id, err)
					continue
				}

				if err := search.SaveAttachments(email, "./data/attachments", from); err != nil {
					log.Printf("Error saving attachments for %s: %v", id, err)
				}

				if _, err := db.SaveEmail(email); err != nil {
					log.Printf("Error saving email %s: %v", id, err)
					continue
				}

				fetched++
			}

			if skipped > 0 {
				fmt.Printf("  Skipped %d (already fetched)\n", skipped)
			}
			if fetched > 0 {
				fmt.Printf("  Fetched %d new emails\n", fetched)
			}
		}
	}

	fmt.Println("\nFetch complete.")
}

// --- parse command ---

func runParse() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	stmtDB, err := store.NewStatementStore("data/statements.db")
	if err != nil {
		log.Fatalf("Failed to open statement database: %v", err)
	}
	defer stmtDB.Close()

	totalParsed := 0
	totalSkipped := 0

	for _, fs := range cfg.FinancialSources {
		parseFn, err := parser.GetStatementParser(fs.Parser)
		if err != nil {
			log.Fatalf("Financial source %q: %v", fs.Name, err)
		}

		password := ""
		if fs.Password != "" {
			password = cfg.PDFPasswords[fs.Password]
		}

		files, err := filepath.Glob(fs.Glob)
		if err != nil {
			log.Printf("Error globbing %s: %v", fs.Glob, err)
			continue
		}

		for _, f := range files {
			lower := strings.ToLower(f)
			if !strings.HasSuffix(lower, ".pdf") && !strings.HasSuffix(lower, ".csv") {
				continue
			}

			exists, err := stmtDB.StatementExists(f)
			if err != nil {
				log.Printf("Error checking %s: %v", f, err)
				continue
			}
			if exists {
				totalSkipped++
				continue
			}

			fmt.Printf("Parsing [%s] %s...\n", fs.Name, filepath.Base(f))
			stmt, err := parseFn(f, password)
			if err != nil {
				log.Printf("Error parsing %s: %v", f, err)
				continue
			}

			if err := stmtDB.SaveStatement(stmt); err != nil {
				log.Printf("Error saving %s: %v", f, err)
				continue
			}

			fmt.Printf("  → %d transactions (%s to %s)\n",
				len(stmt.Transactions),
				stmt.PeriodFrom.Format("2006-01-02"),
				stmt.PeriodTo.Format("2006-01-02"))
			totalParsed++
		}
	}

	// Parse dashboard invoices
	invoicesParsed := 0
	invoicesSkipped := 0

	for _, ds := range cfg.DashboardSources {
		parseFn, err := parser.GetInvoiceParser(ds.Parser)
		if err != nil {
			log.Fatalf("Dashboard source %q: %v", ds.Name, err)
		}

		files, err := filepath.Glob(ds.Glob)
		if err != nil {
			log.Printf("Error globbing %s: %v", ds.Glob, err)
			continue
		}

		for _, f := range files {
			if !strings.HasSuffix(strings.ToLower(f), ".pdf") {
				continue
			}

			fmt.Printf("Parsing [%s] %s...\n", ds.Name, filepath.Base(f))
			inv, err := parseFn(f)
			if err != nil {
				log.Printf("Error parsing %s: %v", f, err)
				continue
			}

			exists, err := stmtDB.InvoiceExists(inv.InvoiceNumber)
			if err != nil {
				log.Printf("Error checking %s: %v", f, err)
				continue
			}
			if exists {
				invoicesSkipped++
				continue
			}

			if err := stmtDB.SaveInvoice(inv); err != nil {
				log.Printf("Error saving %s: %v", f, err)
				continue
			}

			fmt.Printf("  → %s: %s%.2f (%s to %s)\n",
				inv.InvoiceNumber, currencySymbol(inv.Currency), inv.Amount,
				inv.PeriodFrom.Format("2006-01-02"),
				inv.PeriodTo.Format("2006-01-02"))
			invoicesParsed++
		}
	}

	fmt.Printf("\nParse complete. Statements: %d parsed, %d skipped. Invoices: %d parsed, %d skipped.\n",
		totalParsed, totalSkipped, invoicesParsed, invoicesSkipped)
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

// --- recon command ---

func runRecon() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	services, err := config.BuildServices(cfg)
	if err != nil {
		log.Fatalf("Failed to build services: %v", err)
	}

	emailDB, err := store.New("data/emails.db")
	if err != nil {
		log.Fatalf("Failed to open email database: %v", err)
	}
	defer emailDB.Close()

	stmtDB, err := store.NewStatementStore("data/statements.db")
	if err != nil {
		log.Fatalf("Failed to open statement database: %v", err)
	}
	defer stmtDB.Close()

	reconDB, err := store.NewReconStore("data/reconciliation.db")
	if err != nil {
		log.Fatalf("Failed to open reconciliation database: %v", err)
	}
	defer reconDB.Close()

	fmt.Println("Running reconciliation...")

	// Clear stale results before re-running
	if err := reconDB.Clear(); err != nil {
		log.Fatalf("Failed to clear reconciliation db: %v", err)
	}

	results, err := recon.RunRecon(services, emailDB, stmtDB)
	if err != nil {
		log.Fatalf("Reconciliation failed: %v", err)
	}

	// Save to DB
	saved, updated := 0, 0
	for _, r := range results {
		row := &store.ReconRow{
			Service:        r.Service,
			Status:         r.Status,
			Date:           r.Date,
			SourceAmount:   r.SourceAmount,
			DestAmount:     r.DestAmount,
			Description:    r.Description,
			Account:        r.Account,
			Source:         r.Source,
			Destination:    r.Destination,
			AttachmentPath: r.AttachmentPath,
			DestRef:        r.DestRef,
			Notes:          r.Notes,
			AmountINR:      r.AmountINR,
		}
		if err := reconDB.Upsert(row); err != nil {
			log.Printf("Error saving result: %v", err)
			continue
		}
		saved++
	}
	if updated > 0 {
		fmt.Printf("Saved %d results to reconciliation.db (%d updated)\n", saved, updated)
	} else {
		fmt.Printf("Saved %d results to reconciliation.db\n", saved)
	}

	// Print summary
	recon.PrintSummary(services, results)
}

// --- package command ---

func runPackage() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	reconDB, err := store.NewReconStore("data/reconciliation.db")
	if err != nil {
		log.Fatalf("Failed to open reconciliation database: %v", err)
	}
	defer reconDB.Close()

	fmt.Println("Packaging reimbursement...")
	if err := recon.PackageFromDB(reconDB, cfg.PDFPasswords); err != nil {
		log.Fatalf("Packaging failed: %v", err)
	}

	fmt.Println("\nPackaging complete.")
}
