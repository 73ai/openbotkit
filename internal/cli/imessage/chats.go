package imessage

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/73ai/openbotkit/config"
	imsrc "github.com/73ai/openbotkit/source/imessage"
	"github.com/spf13/cobra"
)

var chatsCmd = &cobra.Command{
	Use:   "chats",
	Short: "Query stored iMessage chats",
}

var chatsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List stored chats",
	Example: `  obk imessage chats list
  obk imessage chats list --limit 20 --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		db, err := openIMessageDB(cfg)
		if err != nil {
			return err
		}
		defer db.Close()

		limit, _ := cmd.Flags().GetInt("limit")
		jsonOut, _ := cmd.Flags().GetBool("json")

		chats, err := imsrc.ListChats(db, imsrc.ListOptions{
			Limit: limit,
		})
		if err != nil {
			return fmt.Errorf("list chats: %w", err)
		}

		if jsonOut {
			return json.NewEncoder(os.Stdout).Encode(chats)
		}

		if len(chats) == 0 {
			fmt.Println("No chats found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "GUID\tNAME\tSERVICE\tPARTICIPANTS\tGROUP")
		for _, c := range chats {
			name := c.DisplayName
			if name == "" {
				name = "-"
			}
			group := "no"
			if c.IsGroup {
				group = "yes"
			}
			participants := strings.Join(c.Participants, ", ")
			if len(participants) > 40 {
				participants = participants[:37] + "..."
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				truncateGUID(c.GUID), name, c.ServiceName, participants, group)
		}
		return w.Flush()
	},
}

func init() {
	chatsListCmd.Flags().Int("limit", 50, "Maximum number of results")
	chatsListCmd.Flags().Bool("json", false, "Output as JSON")

	chatsCmd.AddCommand(chatsListCmd)
}
