package imessage

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/priyanshujain/openbotkit/config"
	imsrc "github.com/priyanshujain/openbotkit/source/imessage"
	"github.com/spf13/cobra"
)

var messagesCmd = &cobra.Command{
	Use:   "messages",
	Short: "Query stored iMessage messages",
}

var messagesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List stored messages with optional filters",
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

		chatGUID, _ := cmd.Flags().GetString("chat")
		limit, _ := cmd.Flags().GetInt("limit")
		jsonOut, _ := cmd.Flags().GetBool("json")

		msgs, err := imsrc.ListMessages(db, imsrc.ListOptions{
			ChatGUID: chatGUID,
			Limit:    limit,
		})
		if err != nil {
			return fmt.Errorf("list messages: %w", err)
		}

		if jsonOut {
			return json.NewEncoder(os.Stdout).Encode(msgs)
		}

		if len(msgs) == 0 {
			fmt.Println("No messages found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "GUID\tSENDER\tDATE\tTEXT")
		for _, m := range msgs {
			text := m.Text
			if len(text) > 60 {
				text = text[:57] + "..."
			}
			sender := m.SenderID
			if m.IsFromMe {
				sender = "me"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				truncateGUID(m.GUID), sender,
				m.Date.Format("2006-01-02 15:04"), text)
		}
		return w.Flush()
	},
}

var messagesGetCmd = &cobra.Command{
	Use:   "get <guid>",
	Short: "Show full details of a stored message",
	Args:  cobra.ExactArgs(1),
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

		jsonOut, _ := cmd.Flags().GetBool("json")

		msg, err := imsrc.GetMessage(db, args[0])
		if err != nil {
			return fmt.Errorf("get message: %w", err)
		}

		if jsonOut {
			return json.NewEncoder(os.Stdout).Encode(msg)
		}

		fmt.Printf("GUID:       %s\n", msg.GUID)
		fmt.Printf("Chat:       %s\n", msg.ChatGUID)
		fmt.Printf("Sender:     %s\n", msg.SenderID)
		fmt.Printf("From me:    %v\n", msg.IsFromMe)
		fmt.Printf("Date:       %s\n", msg.Date.Format("2006-01-02 15:04:05"))
		fmt.Printf("Chat name:  %s\n", msg.ChatDisplayName)
		if msg.AttachmentsJSON != "" {
			fmt.Printf("Attachments: %s\n", msg.AttachmentsJSON)
		}
		fmt.Printf("\n--- Text ---\n%s\n", msg.Text)
		return nil
	},
}

var messagesSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search messages by text content",
	Args:  cobra.ExactArgs(1),
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

		jsonOut, _ := cmd.Flags().GetBool("json")
		limit, _ := cmd.Flags().GetInt("limit")

		msgs, err := imsrc.SearchMessages(db, args[0], limit)
		if err != nil {
			return fmt.Errorf("search messages: %w", err)
		}

		if jsonOut {
			return json.NewEncoder(os.Stdout).Encode(msgs)
		}

		if len(msgs) == 0 {
			fmt.Println("No messages found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "GUID\tSENDER\tDATE\tTEXT")
		for _, m := range msgs {
			text := m.Text
			if len(text) > 60 {
				text = text[:57] + "..."
			}
			sender := m.SenderID
			if m.IsFromMe {
				sender = "me"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				truncateGUID(m.GUID), sender,
				m.Date.Format("2006-01-02 15:04"), text)
		}
		return w.Flush()
	},
}

func init() {
	messagesListCmd.Flags().String("chat", "", "Filter by chat GUID")
	messagesListCmd.Flags().Int("limit", 50, "Maximum number of results")
	messagesListCmd.Flags().Bool("json", false, "Output as JSON")

	messagesGetCmd.Flags().Bool("json", false, "Output as JSON")

	messagesSearchCmd.Flags().Bool("json", false, "Output as JSON")
	messagesSearchCmd.Flags().Int("limit", 50, "Maximum number of results")

	messagesCmd.AddCommand(messagesListCmd)
	messagesCmd.AddCommand(messagesGetCmd)
	messagesCmd.AddCommand(messagesSearchCmd)
}

func truncateGUID(guid string) string {
	if len(guid) > 30 {
		return "..." + guid[len(guid)-25:]
	}
	return guid
}
