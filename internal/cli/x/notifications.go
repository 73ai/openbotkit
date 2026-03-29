package x

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/73ai/openbotkit/source/twitter"
	"github.com/73ai/openbotkit/source/twitter/client"
	"github.com/spf13/cobra"
)

var notificationsCmd = &cobra.Command{
	Use:     "notifications",
	Short:   "Show mentions and notifications",
	Example: "  obk x notifications\n  obk x notifications --limit 10 --json",
	RunE: func(cmd *cobra.Command, args []string) error {
		limit, _ := cmd.Flags().GetInt("limit")
		jsonOut, _ := cmd.Flags().GetBool("json")

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
		raw, err := xClient.Notifications(ctx, limit, "")
		if err != nil {
			return fmt.Errorf("fetch notifications: %w", err)
		}

		tweets, _, err := twitter.ParseNotifications(raw)
		if err != nil {
			return fmt.Errorf("parse notifications: %w", err)
		}

		if len(tweets) == 0 {
			fmt.Println("No notifications found.")
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
	notificationsCmd.Flags().Int("limit", 20, "Number of notifications")
	notificationsCmd.Flags().Bool("json", false, "Output as JSON")
}
