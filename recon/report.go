package recon

import (
	"fmt"
	"strconv"
)

// PrintSummary prints a reconciliation summary to stdout.
func PrintSummary(services []Service, results []ReconResult) {
	if len(results) == 0 {
		fmt.Println("No reimbursable transactions found.")
		return
	}

	// Count by status
	reconciled := 0
	unreconciled := 0
	for _, r := range results {
		if r.Status == StatusReconciled {
			reconciled++
		} else {
			unreconciled++
		}
	}

	// Total INR (only reconciled)
	var totalINR float64
	byService := make(map[string]float64)
	byServiceCount := make(map[string]int)

	for _, r := range results {
		byServiceCount[r.Service]++
		if r.Status == StatusReconciled {
			amt, err := strconv.ParseFloat(r.DestAmount, 64)
			if err == nil {
				totalINR += amt
				byService[r.Service] += amt
			}
		}
	}

	fmt.Println("\n=== Reimbursement Report Summary ===")
	fmt.Printf("Total transactions: %d\n", len(results))
	fmt.Printf("  Reconciled:   %d\n", reconciled)
	fmt.Printf("  Unreconciled: %d\n", unreconciled)
	fmt.Println()

	// Per-service breakdown
	fmt.Println("Per-service breakdown:")
	// Maintain order from services list
	for _, svc := range services {
		count := byServiceCount[svc.Name]
		if count == 0 {
			continue
		}
		inr := byService[svc.Name]
		if inr > 0 {
			fmt.Printf("  %-20s %d txns  ₹%.2f (reconciled)\n", svc.Name, count, inr)
		} else {
			fmt.Printf("  %-20s %d txns  (unreconciled)\n", svc.Name, count)
		}
	}

	fmt.Printf("\nTotal reconciled INR: ₹%.2f\n", totalINR)

	// List unreconciled items
	if unreconciled > 0 {
		fmt.Println("\n--- Unreconciled (need manual INR amount) ---")
		for _, r := range results {
			if r.Status == StatusUnreconciled {
				src := r.SourceAmount
				if src == "" {
					src = "?"
				}
				fmt.Printf("  %s | %s | %s | %s | %s\n",
					r.Date.Format("2006-01-02"), r.Service, r.Description, src, r.Account)
			}
		}
	}
}
