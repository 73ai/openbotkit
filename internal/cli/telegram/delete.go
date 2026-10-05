package telegram

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/73ai/openbotkit/config"
	tgsrc "github.com/73ai/openbotkit/source/telegram"
	"github.com/73ai/openbotkit/store"
	"github.com/charmbracelet/x/term"
	"github.com/gotd/td/tg"
	"github.com/spf13/cobra"
)

var chatsDeleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   "Delete a chat from your Telegram account (manual CLI only)",
	Long:    "Delete a private chat and its history for your account. For groups and channels, leave the chat and remove its local data. Requires terminal confirmation; unavailable to agents.",
	Example: "  obk telegram chats delete --chat 123456789\n  obk telegram chats delete --chat 123456789 --revoke",
	Args:    cobra.NoArgs,
	RunE:    func(cmd *cobra.Command, args []string) error { return runDelete(cmd, true) },
}

var messagesDeleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   "Delete specific Telegram messages (manual CLI only)",
	Long:    "Delete messages by Telegram message ID in a specific chat. Defaults to your account only for private chats. Group and channel messages require --revoke because deletion affects everyone. Requires terminal confirmation; unavailable to agents.",
	Example: "  obk telegram messages delete --chat 123456789 --ids 42,43\n  obk telegram messages delete --chat 123456789 --ids 42 --revoke",
	Args:    cobra.NoArgs,
	RunE:    func(cmd *cobra.Command, args []string) error { return runDelete(cmd, false) },
}

func runDelete(cmd *cobra.Command, wholeChat bool) error {
	if os.Getenv("OBK_AGENT") != "" {
		return fmt.Errorf("Telegram deletion is only available in the manual CLI")
	}
	if !term.IsTerminal(os.Stdin.Fd()) {
		return fmt.Errorf("Telegram deletion requires an interactive terminal")
	}
	chatID, _ := cmd.Flags().GetInt64("chat")
	if chatID == 0 {
		return fmt.Errorf("--chat must be a nonzero Telegram chat ID")
	}
	revoke, _ := cmd.Flags().GetBool("revoke")
	var ids []int
	if !wholeChat {
		ids, _ = cmd.Flags().GetIntSlice("ids")
		if len(ids) == 0 || len(ids) > 100 {
			return fmt.Errorf("--ids requires between 1 and 100 Telegram message IDs")
		}
		seen := make(map[int]bool, len(ids))
		for _, id := range ids {
			if id <= 0 || int64(id) > 2147483647 || seen[id] {
				return fmt.Errorf("message IDs must be distinct positive 32-bit integers")
			}
			seen[id] = true
		}
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if err := requireSession(cfg); err != nil {
		return err
	}
	db, err := openTelegramDB(cfg)
	if err != nil {
		return err
	}
	defer db.Close()
	chat, err := tgsrc.GetChat(db, chatID)
	if err != nil {
		return fmt.Errorf("look up chat: %w", err)
	}
	if chat == nil {
		return fmt.Errorf("unknown chat %d; run 'obk telegram chats list' to find a synced chat", chatID)
	}
	peer, err := tgsrc.InputPeerFor(chat)
	if err != nil {
		return err
	}
	kind, rawID := tgsrc.SplitChatID(chatID)
	if wholeChat && kind != tgsrc.PeerUser && revoke {
		return fmt.Errorf("--revoke is only supported when deleting private chats")
	}
	if !wholeChat && kind != tgsrc.PeerUser && !revoke {
		return fmt.Errorf("group and channel message deletion affects everyone; specify --revoke")
	}

	scope := "for your account only"
	if revoke {
		scope = "for everyone"
	}
	if wholeChat {
		if kind == tgsrc.PeerUser {
			fmt.Fprintf(cmd.OutOrStdout(), "Delete private chat %q (%d) and all its history %s.\n", chat.Title, chatID, scope)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Leave chat %q (%d) and remove its local messages. Other members keep their history.\n", chat.Title, chatID)
		}
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Delete messages %v in chat %q (%d) %s.\n", ids, chat.Title, chatID, scope)
	}
	confirmation := fmt.Sprintf("delete %d", chatID)
	fmt.Fprintf(cmd.OutOrStdout(), "Type %q to confirm: ", confirmation)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil || strings.TrimSpace(line) != confirmation {
		return fmt.Errorf("deletion cancelled")
	}

	client, err := newClient(cfg, db)
	if err != nil {
		return err
	}
	err = client.Run(cmd.Context(), func(ctx context.Context) error {
		api := client.API()
		if !wholeChat {
			return deleteRemoteMessages(ctx, api, db, chatID, peer, ids, revoke)
		}
		switch kind {
		case tgsrc.PeerChannel:
			p := peer.(*tg.InputPeerChannel)
			if _, err := api.ChannelsLeaveChannel(ctx, &tg.InputChannel{ChannelID: p.ChannelID, AccessHash: p.AccessHash}); err != nil {
				return fmt.Errorf("leave channel: %w", err)
			}
		case tgsrc.PeerGroup:
			if _, err := api.MessagesDeleteChatUser(ctx, &tg.MessagesDeleteChatUserRequest{ChatID: rawID, UserID: &tg.InputUserSelf{}}); err != nil {
				return fmt.Errorf("leave group: %w", err)
			}
			fallthrough
		case tgsrc.PeerUser:
			for {
				result, err := api.MessagesDeleteHistory(ctx, &tg.MessagesDeleteHistoryRequest{Peer: peer, Revoke: revoke})
				if err != nil {
					if kind == tgsrc.PeerGroup {
						return fmt.Errorf("left the group, but deleting its history failed: %w", err)
					}
					return fmt.Errorf("delete history: %w", err)
				}
				if result.Offset == 0 {
					break
				}
			}
		}
		return removeLocalChat(db, chatID)
	})
	if err != nil {
		return err
	}
	if wholeChat {
		fmt.Fprintf(cmd.OutOrStdout(), "Chat deleted: %d\n", chatID)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Messages deleted: %v in chat %d\n", ids, chatID)
	}
	return nil
}

func deleteRemoteMessages(ctx context.Context, api *tg.Client, db *store.DB, chatID int64, peer tg.InputPeerClass, ids []int, revoke bool) error {
	if p, ok := peer.(*tg.InputPeerChannel); ok {
		_, err := api.ChannelsDeleteMessages(ctx, &tg.ChannelsDeleteMessagesRequest{
			Channel: &tg.InputChannel{ChannelID: p.ChannelID, AccessHash: p.AccessHash}, ID: ids,
		})
		if err != nil {
			return fmt.Errorf("delete channel messages: %w", err)
		}
	} else {
		inputs := make([]tg.InputMessageClass, len(ids))
		for i, id := range ids {
			inputs[i] = &tg.InputMessageID{ID: id}
		}
		result, err := api.MessagesGetMessages(ctx, inputs)
		if err != nil {
			return fmt.Errorf("verify message chat: %w", err)
		}
		messages, ok := result.(interface{ GetMessages() []tg.MessageClass })
		if !ok {
			return fmt.Errorf("could not verify message chat")
		}
		verified := make(map[int]bool, len(ids))
		for _, raw := range messages.GetMessages() {
			message, ok := raw.(interface {
				GetID() int
				GetPeerID() tg.PeerClass
			})
			if !ok {
				continue
			}
			id, ok := tgsrc.ChatIDFromPeer(message.GetPeerID())
			if !ok || id != chatID {
				return fmt.Errorf("message %d does not belong to chat %d", message.GetID(), chatID)
			}
			verified[message.GetID()] = true
		}
		for _, id := range ids {
			if !verified[id] {
				return fmt.Errorf("message %d was not found in chat %d", id, chatID)
			}
		}
		if _, err := api.MessagesDeleteMessages(ctx, &tg.MessagesDeleteMessagesRequest{ID: ids, Revoke: revoke}); err != nil {
			return fmt.Errorf("delete messages: %w", err)
		}
	}
	if err := tgsrc.DeleteMessages(db, chatID, ids); err != nil {
		return fmt.Errorf("messages deleted on Telegram, but local cleanup failed: %w", err)
	}
	return nil
}

func removeLocalChat(db *store.DB, chatID int64) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("chat deleted on Telegram, but local cleanup failed: %w", err)
	}
	defer tx.Rollback()
	for _, table := range []string{"telegram_messages", "telegram_sync_state", "telegram_chats"} {
		if _, err := tx.Exec(db.Rebind("DELETE FROM "+table+" WHERE chat_id = ?"), chatID); err != nil {
			return fmt.Errorf("chat deleted on Telegram, but local cleanup failed: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("chat deleted on Telegram, but local cleanup failed: %w", err)
	}
	return nil
}

func init() {
	for _, cmd := range []*cobra.Command{chatsDeleteCmd, messagesDeleteCmd} {
		cmd.Flags().Int64("chat", 0, "Exact Telegram chat ID (required)")
		cmd.Flags().Bool("revoke", false, "Delete for everyone instead of only your account")
		cmd.MarkFlagRequired("chat")
	}
	messagesDeleteCmd.Flags().IntSlice("ids", nil, "Comma-separated Telegram message IDs, up to 100 (required)")
	messagesDeleteCmd.MarkFlagRequired("ids")
	chatsCmd.AddCommand(chatsDeleteCmd)
	messagesCmd.AddCommand(messagesDeleteCmd)
}
