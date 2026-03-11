package websearch

import "github.com/spf13/cobra"

var Cmd = &cobra.Command{
	Use:   "websearch",
	Short: "Search the web and fetch web pages",
}

func init() {
	Cmd.AddCommand(searchCmd)
	Cmd.AddCommand(fetchCmd)
}
