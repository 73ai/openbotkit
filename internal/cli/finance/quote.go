package finance

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	finsrc "github.com/priyanshujain/openbotkit/source/finance"
	"github.com/spf13/cobra"
)

var quoteCmd = &cobra.Command{
	Use:   "quote [symbols...]",
	Short: "Get stock quotes",
	Long:  "Look up current prices for one or more stock symbols (e.g., AAPL GOOGL MSFT).",
	Example: `  obk finance quote AAPL
  obk finance quote AAPL GOOGL MSFT
  obk finance quote TSLA --json`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := finsrc.NewClient()
		quotes, err := client.Quote(cmd.Context(), args...)
		if err != nil {
			return fmt.Errorf("fetch quotes: %w", err)
		}

		if len(quotes) == 0 {
			fmt.Println("No results found.")
			return nil
		}

		jsonOut, _ := cmd.Flags().GetBool("json")
		if jsonOut {
			return json.NewEncoder(os.Stdout).Encode(quotes)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "SYMBOL\tNAME\tPRICE\tCHANGE\tCURRENCY\tEXCHANGE")
		for _, q := range quotes {
			change := fmt.Sprintf("%+.2f (%+.2f%%)",
				q.RegularMarketChange, q.RegularMarketChangePercent)
			fmt.Fprintf(w, "%s\t%s\t%.2f\t%s\t%s\t%s\n",
				q.Symbol, q.ShortName, q.RegularMarketPrice,
				change, q.Currency, q.Exchange)
		}
		return w.Flush()
	},
}

func init() {
	quoteCmd.Flags().Bool("json", false, "Output as JSON")
}
