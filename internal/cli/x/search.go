package x

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/source/twitter"
	"github.com/73ai/openbotkit/source/twitter/client"
	"github.com/73ai/openbotkit/store"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:     "search [query]",
	Short:   "Search posts",
	Example: "  obk x search \"golang\"\n  obk x search \"from:elonmusk\" --limit 10",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		limit, _ := cmd.Flags().GetInt("limit")
		jsonOut, _ := cmd.Flags().GetBool("json")
		localOnly, _ := cmd.Flags().GetBool("local")
		query := args[0]

		if localOnly {
			return searchLocal(query, limit, jsonOut)
		}
		return searchRemote(query, limit, jsonOut)
	},
}

func searchLocal(query string, limit int, jsonOut bool) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	db, err := store.Open(store.Config{
		Driver: cfg.X.Storage.Driver,
		DSN:    cfg.XDataDSN(),
	})
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	tweets, err := twitter.SearchTweets(db, query, limit)
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}
	return printTweets(tweets, jsonOut)
}

func searchRemote(query string, limit int, jsonOut bool) error {
	session, err := client.LoadSession()
	if err != nil {
		return fmt.Errorf("not authenticated — run 'obk x auth login' first")
	}
	xClient, err := client.NewClient(session, client.DefaultEndpointsPath())
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}

	if limit <= 0 {
		limit = 20
	}

	ctx := context.Background()
	raw, err := xClient.Search(ctx, query, limit, "")
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	tweets, _, err := twitter.ParseSearchResponse(raw)
	if err != nil {
		return fmt.Errorf("parse search results: %w", err)
	}
	return printTweets(tweets, jsonOut)
}

func printTweets(tweets []twitter.Tweet, jsonOut bool) error {
	if len(tweets) == 0 {
		fmt.Println("No results found.")
		return nil
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(tweets)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tUSER\tDATE\tTEXT")
	for _, tw := range tweets {
		text := tw.Text
		if len(text) > 80 {
			text = text[:77] + "..."
		}
		fmt.Fprintf(w, "%s\t@%s\t%s\t%s\n",
			tw.TweetID,
			tw.UserName,
			tw.CreatedAt.Format("2006-01-02 15:04"),
			text,
		)
	}
	return w.Flush()
}

func init() {
	searchCmd.Flags().Int("limit", 20, "Number of results")
	searchCmd.Flags().Bool("json", false, "Output as JSON")
	searchCmd.Flags().Bool("local", false, "Search local database only")
}
