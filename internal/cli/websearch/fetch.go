package websearch

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/priyanshujain/openbotkit/config"
	wssrc "github.com/priyanshujain/openbotkit/source/websearch"
	"github.com/spf13/cobra"
)

var fetchCmd = &cobra.Command{
	Use:   "fetch [url]",
	Short: "Fetch and read a web page",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		format, _ := cmd.Flags().GetString("format")
		maxLength, _ := cmd.Flags().GetInt("max-length")

		ws := wssrc.New(wssrc.Config{WebSearch: cfg.WebSearch})
		result, err := ws.Fetch(cmd.Context(), args[0], wssrc.FetchOptions{
			Format:    format,
			MaxLength: maxLength,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		return json.NewEncoder(os.Stdout).Encode(result)
	},
}

func init() {
	fetchCmd.Flags().StringP("format", "f", "markdown", "Output format (markdown, text)")
	fetchCmd.Flags().Int("max-length", 100000, "Maximum content length")
}
