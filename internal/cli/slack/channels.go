package slack

import (
	"encoding/json"
	"fmt"

	slacksrc "github.com/priyanshujain/openbotkit/source/slack"
	"github.com/spf13/cobra"
)

var channelsCmd = &cobra.Command{
	Use:   "channels",
	Short: "List Slack channels",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := loadClient()
		if err != nil {
			return err
		}

		channels, err := client.ConversationsList(cmd.Context())
		if err != nil {
			return fmt.Errorf("list channels: %w", err)
		}

		fmt.Printf("Found %d channels:\n\n", len(channels))
		for _, ch := range channels {
			data, _ := json.MarshalIndent(ch, "", "  ")
			fmt.Println(string(data))
		}
		return nil
	},
}

var readCmd = &cobra.Command{
	Use:   "read <channel>",
	Short: "Read messages from a Slack channel",
	Args:  cobra.ExactArgs(1),
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
	readCmd.Flags().IntP("limit", "l", 20, "Number of messages to fetch")
}
