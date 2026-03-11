package finance

import "github.com/spf13/cobra"

var Cmd = &cobra.Command{
	Use:   "finance",
	Short: "Look up stock prices and exchange rates",
}

func init() {
	Cmd.AddCommand(quoteCmd)
	Cmd.AddCommand(rateCmd)
}
