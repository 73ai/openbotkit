package websearch

import "github.com/spf13/cobra"

var Cmd = &cobra.Command{
	Use:   "websearch",
	Short: "Manage web search data source",
}

func init() {
	Cmd.AddCommand(searchCmd)
	Cmd.AddCommand(fetchCmd)
	Cmd.AddCommand(newsCmd)
	Cmd.AddCommand(backendsCmd)
	Cmd.AddCommand(historyCmd)
	Cmd.AddCommand(cacheCmd)
}
