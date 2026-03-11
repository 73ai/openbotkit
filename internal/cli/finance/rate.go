package finance

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	finsrc "github.com/priyanshujain/openbotkit/source/finance"
	"github.com/spf13/cobra"
)

var rateCmd = &cobra.Command{
	Use:   "rate [from] [to]",
	Short: "Get exchange rate",
	Long:  "Look up the current exchange rate between two currencies (e.g., USD INR).",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		from := strings.ToUpper(args[0])
		to := strings.ToUpper(args[1])
		symbol := from + to + "=X"

		client := finsrc.NewClient()
		quotes, err := client.Quote(cmd.Context(), symbol)
		if err != nil {
			return fmt.Errorf("fetch rate: %w", err)
		}

		if len(quotes) == 0 {
			return fmt.Errorf("no rate found for %s/%s", from, to)
		}

		jsonOut, _ := cmd.Flags().GetBool("json")
		if jsonOut {
			return json.NewEncoder(os.Stdout).Encode(quotes[0])
		}

		q := quotes[0]
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "PAIR\tRATE\tCHANGE")
		change := fmt.Sprintf("%+.2f (%+.2f%%)",
			q.RegularMarketChange, q.RegularMarketChangePercent)
		fmt.Fprintf(w, "%s/%s\t%.4f\t%s\n", from, to, q.RegularMarketPrice, change)
		return w.Flush()
	},
}

func init() {
	rateCmd.Flags().Bool("json", false, "Output as JSON")
}
