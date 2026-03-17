package slack

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	slacksrc "github.com/73ai/openbotkit/source/slack"
	"github.com/spf13/cobra"
)

var channelsCmd = &cobra.Command{
	Use:   "channels",
	Short: "List Slack channels",
	Example: `  obk slack channels
  obk slack channels --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := loadClient()
		if err != nil {
			return err
		}

		channels, err := client.ConversationsList(cmd.Context())
		if err != nil {
			return fmt.Errorf("list channels: %w", err)
		}

		jsonOut, _ := cmd.Flags().GetBool("json")
		if jsonOut {
			return json.NewEncoder(os.Stdout).Encode(channels)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tMEMBERS\tTOPIC")
		for _, ch := range channels {
			topic := ""
			if ch.Topic != nil {
				topic = ch.Topic.Value
			}
			fmt.Fprintf(w, "%s\t%s\t%d\t%s\n", ch.ID, ch.Name, ch.NumMembers, topic)
		}
		return w.Flush()
	},
}

var readCmd = &cobra.Command{
	Use:   "read <channel>",
	Short: "Read messages from a Slack channel",
	Example: `  obk slack read general
  obk slack read C01ABC23DEF --limit 50`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := loadClient()
		if err != nil {
			return err
		}

		limit, _ := cmd.Flags().GetInt("limit")
		if limit <= 0 {
			limit = 20
		}

		channelRef := args[0]
		channelID, err := client.ResolveChannel(cmd.Context(), channelRef)
		if err != nil {
			return fmt.Errorf("resolve channel: %w", err)
		}

		msgs, err := client.ConversationsHistory(cmd.Context(), channelID, slacksrc.HistoryOptions{Limit: limit})
		if err != nil {
			return fmt.Errorf("read channel: %w", err)
		}

		for _, msg := range msgs {
			data, _ := json.MarshalIndent(msg, "", "  ")
			fmt.Println(string(data))
			fmt.Println()
		}
		return nil
	},
}

func init() {
	channelsCmd.Flags().Bool("json", false, "Output as JSON")
	readCmd.Flags().IntP("limit", "l", 20, "Number of messages to fetch")
}
