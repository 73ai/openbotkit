package websearch

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/priyanshujain/openbotkit/config"
	wssrc "github.com/priyanshujain/openbotkit/source/websearch"
	"github.com/priyanshujain/openbotkit/store"
	"github.com/spf13/cobra"
)

var fetchCmd = &cobra.Command{
	Use:   "fetch [url]",
	Short: "Fetch and read a web page",
	Example: `  obk websearch fetch "https://example.com/article"
  obk websearch fetch "https://example.com" -f text
  obk websearch fetch "https://example.com" --no-cache`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		format, _ := cmd.Flags().GetString("format")
		maxLength, _ := cmd.Flags().GetInt("max-length")
		noCache, _ := cmd.Flags().GetBool("no-cache")

		var opts []wssrc.Option
		db, err := store.Open(store.SQLiteConfig(cfg.WebSearchDataDSN()))
		if err == nil {
			defer db.Close()
			if err := wssrc.Migrate(db); err == nil {
				opts = append(opts, wssrc.WithDB(db))
			}
		}

		ws := wssrc.New(wssrc.Config{WebSearch: cfg.WebSearch}, opts...)
		result, err := ws.Fetch(cmd.Context(), args[0], wssrc.FetchOptions{
			Format:    format,
			MaxLength: maxLength,
			NoCache:   noCache,
		})
		if err != nil {
			return err
		}

		return json.NewEncoder(os.Stdout).Encode(result)
	},
}

func init() {
	fetchCmd.Flags().StringP("format", "f", "markdown", "Output format (markdown, text)")
	fetchCmd.Flags().Int("max-length", 100000, "Maximum content length")
	fetchCmd.Flags().Bool("no-cache", false, "Bypass result cache")
}
