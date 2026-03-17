package cli

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"text/tabwriter"
	"time"

	"github.com/73ai/openbotkit/config"
	usagesrc "github.com/73ai/openbotkit/source/usage"
	"github.com/73ai/openbotkit/store"
	"github.com/spf13/cobra"
)

var usageCmd = &cobra.Command{
	Use:   "usage",
	Short: "Show LLM token usage and estimated costs",
	Example: `  obk usage
  obk usage --since 2025-03-01 --json`,
	RunE:  runUsageDaily,
}

var usageDailyCmd = &cobra.Command{
	Use:   "daily",
	Short: "Show daily usage breakdown",
	Example: `  obk usage daily --since 2025-03-01
  obk usage daily --model claude-sonnet-4-6 --json`,
	RunE:  runUsageDaily,
}

var usageMonthlyCmd = &cobra.Command{
	Use:   "monthly",
	Short: "Show monthly usage breakdown",
	Example: `  obk usage monthly
  obk usage monthly --json`,
	RunE:  runUsageMonthly,
}

var (
	usageSince string
	usageUntil string
	usageModel string
	usageJSON  bool
)

func init() {
	for _, cmd := range []*cobra.Command{usageCmd, usageDailyCmd, usageMonthlyCmd} {
		cmd.Flags().StringVar(&usageSince, "since", "", "Start date (YYYY-MM-DD)")
		cmd.Flags().StringVar(&usageUntil, "until", "", "End date (YYYY-MM-DD)")
		cmd.Flags().StringVar(&usageModel, "model", "", "Filter by model name")
		cmd.Flags().BoolVar(&usageJSON, "json", false, "Output as JSON")
	}
	usageCmd.AddCommand(usageDailyCmd)
	usageCmd.AddCommand(usageMonthlyCmd)
	rootCmd.AddCommand(usageCmd)
}

func runUsageDaily(cmd *cobra.Command, args []string) error {
	return runUsageQuery("daily")
}

func runUsageMonthly(cmd *cobra.Command, args []string) error {
	return runUsageQuery("monthly")
}

func runUsageQuery(groupBy string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	db, err := store.Open(store.Config{
		Driver: cfg.Usage.Storage.Driver,
		DSN:    cfg.UsageDataDSN(),
	})
	if err != nil {
		return fmt.Errorf("open usage db: %w", err)
	}
	defer db.Close()

	if err := usagesrc.Migrate(db); err != nil {
		return fmt.Errorf("migrate usage: %w", err)
	}

	opts := usagesrc.QueryOpts{
		GroupBy: groupBy,
		Model:   usageModel,
	}

	if usageSince != "" {
		t, err := time.Parse("2006-01-02", usageSince)
		if err != nil {
			return fmt.Errorf("invalid --since date: %w", err)
		}
		opts.Since = &t
	} else {
		t := time.Now().AddDate(0, 0, -30)
		opts.Since = &t
	}

	if usageUntil != "" {
		t, err := time.Parse("2006-01-02", usageUntil)
		if err != nil {
			return fmt.Errorf("invalid --until date: %w", err)
		}
		opts.Until = &t
	}

	results, err := usagesrc.Query(db, opts)
	if err != nil {
		return fmt.Errorf("query usage: %w", err)
	}

	if usageJSON {
		return json.NewEncoder(os.Stdout).Encode(results)
	}

	if len(results) == 0 {
		fmt.Println("No usage data found for the selected period.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintf(w, "Date\tModel\tInput\tOutput\tCache Read\tCache Write\tCalls\tEst. Cost\n")
	fmt.Fprintf(w, "----\t-----\t-----\t------\t----------\t-----------\t-----\t---------\n")

	var totalCost float64
	for _, r := range results {
		cost := estimateCost(r)
		totalCost += cost
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%d\t$%.2f\n",
			r.Date, r.Model,
			formatTokens(r.InputTokens), formatTokens(r.OutputTokens),
			formatTokens(r.CacheReadTokens), formatTokens(r.CacheWriteTokens),
			r.CallCount, cost)
	}
	fmt.Fprintf(w, "TOTAL\t\t\t\t\t\t\t$%.2f\n", totalCost)
	w.Flush()

	return nil
}

func formatTokens(n int64) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}

// Pricing per million tokens (input, output, cache_read).
// Cache writes are charged at input rate.
var modelPricing = map[string][3]float64{
	"claude-sonnet-4-6":        {3.0, 15.0, 0.30},
	"claude-sonnet-4-20250514": {3.0, 15.0, 0.30},
	"claude-haiku-4-5":         {0.80, 4.0, 0.08},
	"claude-opus-4-6":          {15.0, 75.0, 1.50},
	"gpt-4o":                   {2.50, 10.0, 1.25},
	"gpt-4o-mini":              {0.15, 0.60, 0.075},
	"gpt-4.1":                  {2.00, 8.00, 0.50},
	"gpt-4.1-mini":             {0.40, 1.60, 0.10},
	"gpt-4.1-nano":             {0.10, 0.40, 0.025},
	"gemini-2.5-pro":           {1.25, 10.0, 0.3125},
	"gemini-2.5-flash":         {0.15, 0.60, 0.0375},
}

func estimateCost(r usagesrc.AggregatedUsage) float64 {
	pricing, ok := modelPricing[r.Model]
	if !ok {
		// Try prefix matching for versioned model names.
		// Use longest match to avoid "gpt-4o" matching "gpt-4o-mini-*".
		bestLen := 0
		for prefix, p := range modelPricing {
			if len(prefix) > bestLen && len(r.Model) >= len(prefix) && r.Model[:len(prefix)] == prefix {
				pricing = p
				bestLen = len(prefix)
				ok = true
			}
		}
	}
	if !ok {
		return 0
	}

	inputRate := pricing[0] / 1_000_000
	outputRate := pricing[1] / 1_000_000
	cacheReadRate := pricing[2] / 1_000_000

	// Non-cached input tokens = total input - cache_read.
	nonCachedInput := r.InputTokens - r.CacheReadTokens
	if nonCachedInput < 0 {
		nonCachedInput = 0
	}

	cost := float64(nonCachedInput)*inputRate +
		float64(r.OutputTokens)*outputRate +
		float64(r.CacheReadTokens)*cacheReadRate +
		float64(r.CacheWriteTokens)*inputRate // cache writes charged at input rate

	return math.Round(cost*100) / 100
}
