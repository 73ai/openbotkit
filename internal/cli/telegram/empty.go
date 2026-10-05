package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"text/tabwriter"

	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/internal/platform"
	tgsrc "github.com/73ai/openbotkit/source/telegram"
	"github.com/spf13/cobra"
)

var chatsEmptyCmd = &cobra.Command{
	Use:   "empty",
	Short: "Verify current private chats with no regular messages using live history",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		if err := requireSession(cfg); err != nil {
			return err
		}
		client, err := newClient(cfg, nil)
		if err != nil {
			return err
		}
		ctx, stop := signal.NotifyContext(cmd.Context(), platform.ShutdownSignals...)
		defer stop()
		var chats []tgsrc.EmptyChat
		if err := client.Run(ctx, func(ctx context.Context) error {
			chats, err = tgsrc.FindEmptyChats(ctx, tgsrc.NewAPIFetcher(client.API()))
			return err
		}); err != nil {
			return err
		}
		jsonOut, _ := cmd.Flags().GetBool("json")
		if jsonOut {
			return json.NewEncoder(os.Stdout).Encode(chats)
		}
		if len(chats) == 0 {
			fmt.Println("No current private chats without regular messages found.")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "CHAT ID\tTITLE\tUSERNAME\tKIND")
		for _, result := range chats {
			fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", result.Chat.ChatID, truncate(result.Chat.Title, 30), result.Chat.Username, result.Kind)
		}
		return w.Flush()
	},
}

func init() {
	chatsEmptyCmd.Flags().Bool("json", false, "Output as JSON")
	chatsCmd.AddCommand(chatsEmptyCmd)
}
