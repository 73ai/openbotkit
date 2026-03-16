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

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search the web for information",
	Example: `  obk websearch search "golang concurrency patterns"
  obk websearch search "Da Nang restaurants" -n 5 -b duckduckgo
  obk websearch search "recent CVEs" -t w -r us-en
  obk websearch search "rust vs go" --no-cache`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		maxResults, _ := cmd.Flags().GetInt("max-results")
		backend, _ := cmd.Flags().GetString("backend")
		timeLimit, _ := cmd.Flags().GetString("time-limit")
		region, _ := cmd.Flags().GetString("region")
		page, _ := cmd.Flags().GetInt("page")
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
		result, err := ws.Search(cmd.Context(), args[0], wssrc.SearchOptions{
			MaxResults: maxResults,
			Backend:    backend,
			TimeLimit:  timeLimit,
			Region:     region,
			Page:       page,
			NoCache:    noCache,
		})
		if err != nil {
			return err
		}

		return json.NewEncoder(os.Stdout).Encode(result)
	},
}

func init() {
	searchCmd.Flags().IntP("max-results", "n", 10, "Maximum number of results")
	searchCmd.Flags().StringP("backend", "b", "auto", "Search backend (auto, duckduckgo, brave, mojeek, yahoo, yandex, google, wikipedia, bing)")
	searchCmd.Flags().StringP("time-limit", "t", "", "Time limit (d=day, w=week, m=month)")
	searchCmd.Flags().StringP("region", "r", "us-en", "Region for search results")
	searchCmd.Flags().IntP("page", "p", 1, "Page number for pagination")
	searchCmd.Flags().Bool("no-cache", false, "Bypass result cache")
}
