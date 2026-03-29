package x

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/source/twitter"
	"github.com/73ai/openbotkit/store"
	"github.com/spf13/cobra"
)

var timelineCmd = &cobra.Command{
	Use:   "timeline",
	Short: "Show posts from local database",
	Example: `  obk x timeline
  obk x timeline --limit 20
  obk x timeline --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
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

		limit, _ := cmd.Flags().GetInt("limit")
		jsonOut, _ := cmd.Flags().GetBool("json")

		tweets, err := twitter.ListTweets(db, twitter.ListOptions{Limit: limit})
		if err != nil {
			return fmt.Errorf("list posts: %w", err)
		}

		if len(tweets) == 0 {
			fmt.Println("No posts found. Run 'obk x sync' first.")
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
	},
}

func init() {
	timelineCmd.Flags().Int("limit", 20, "Number of posts to show")
	timelineCmd.Flags().Bool("json", false, "Output as JSON")
}
