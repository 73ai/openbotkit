package slack

import (
	"encoding/json"
	"fmt"
	"strings"

	slacksrc "github.com/priyanshujain/openbotkit/source/slack"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search Slack messages",
	Example: `  obk slack search "deploy failed"
  obk slack search "from:alice in:#general" --limit 5`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := loadClient()
		if err != nil {
			return err
		}

		limit, _ := cmd.Flags().GetInt("limit")
		if limit <= 0 {
			limit = 10
		}

		query := strings.Join(args, " ")
		result, err := client.SearchMessages(cmd.Context(), query, slacksrc.SearchOptions{Count: limit})
		if err != nil {
			return fmt.Errorf("search: %w", err)
		}

		fmt.Printf("Found %d results (showing page %d of %d):\n\n", result.Total, result.Page, result.Pages)
		for _, msg := range result.Messages {
			data, _ := json.MarshalIndent(msg, "", "  ")
			fmt.Println(string(data))
			fmt.Println()
		}
		return nil
	},
}

func init() {
	searchCmd.Flags().IntP("limit", "l", 10, "Maximum results")
}
