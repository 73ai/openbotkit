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
	"github.com/gotd/td/tg"
	"github.com/spf13/cobra"
)

var chatsDeleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   "Delete a chat from your Telegram account (manual CLI only)",
	Long:    "Delete a private chat and its history for your account. For groups and channels, leave the chat and remove its local data. Requires terminal confirmation; unavailable to agents.",
	Example: "  obk telegram chats delete --chat 123456789\n  obk telegram chats delete --chat 123456789 --revoke",
	Args:    cobra.NoArgs,
	RunE:    runChatDelete,
}

var messagesDeleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   "Delete specific Telegram messages (manual CLI only)",
	Long:    "Delete messages by Telegram message ID in a specific chat. Defaults to your account only. Channel and supergroup deletion always affects everyone and requires --revoke. Requires terminal confirmation; unavailable to agents.",
	Example: "  obk telegram messages delete --chat 123456789 --ids 42,43\n  obk telegram messages delete --chat 123456789 --ids 42 --revoke",
	Args:    cobra.NoArgs,
	RunE:    runMessagesDelete,
}

type deleteTarget struct {
	cfg    *config.Config
	db     *store.DB
	chat   *tgsrc.Chat
	peer   tg.InputPeerClass
	kind   string
	rawID  int64
	revoke bool
}

func runChatDelete(cmd *cobra.Command, args []string) error {
	t, err := loadDeleteTarget(cmd)
	if err != nil {
		return err
	}
	defer t.db.Close()
	if t.kind != tgsrc.PeerUser && t.revoke {
		return fmt.Errorf("--revoke is only supported when deleting private chats")
	}
	if t.kind == tgsrc.PeerUser {
		fmt.Fprintf(cmd.OutOrStdout(), "Delete private chat %q (%d) and all its history %s.\n", t.chat.Title, t.chat.ChatID, scope(t.revoke))
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Leave chat %q (%d) and remove its local messages. Other members keep their history.\n", t.chat.Title, t.chat.ChatID)
	}
	if err := t.run(cmd, func(ctx context.Context, api *tg.Client) error {
		if err := t.deleteRemoteChat(ctx, api); err != nil {
			return err
		}
		if err := tgsrc.DeleteChat(t.db, t.chat.ChatID); err != nil {
			return fmt.Errorf("chat deleted on Telegram, but local cleanup failed: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Chat deleted: %d\n", t.chat.ChatID)
	return nil
}

func runMessagesDelete(cmd *cobra.Command, args []string) error {
	t, err := loadDeleteTarget(cmd)
	if err != nil {
		return err
	}
	defer t.db.Close()
	ids, _ := cmd.Flags().GetIntSlice("ids")
	if len(ids) == 0 || len(ids) > 100 {
		return fmt.Errorf("--ids requires between 1 and 100 Telegram message IDs")
	}
	if t.kind == tgsrc.PeerChannel && !t.revoke {
		return fmt.Errorf("channel and supergroup message deletion affects everyone; specify --revoke")
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Delete messages %v in chat %q (%d) %s.\n", ids, t.chat.Title, t.chat.ChatID, scope(t.revoke))
	if err := t.run(cmd, func(ctx context.Context, api *tg.Client) error {
		if err := t.deleteRemoteMessages(ctx, api, ids); err != nil {
			return err
		}
		if err := tgsrc.DeleteMessages(t.db, t.chat.ChatID, ids); err != nil {
			return fmt.Errorf("messages deleted on Telegram, but local cleanup failed: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Messages deleted: %v in chat %d\n", ids, t.chat.ChatID)
	return nil
}

func scope(revoke bool) string {
	if revoke {
		return "for everyone"
	}
	return "for your account only"
}

func loadDeleteTarget(cmd *cobra.Command) (*deleteTarget, error) {
	if os.Getenv("OBK_AGENT") != "" {
		return nil, fmt.Errorf("Telegram deletion is only available in the manual CLI")
	}
	if info, err := os.Stdin.Stat(); err != nil || info.Mode()&os.ModeCharDevice == 0 {
		return nil, fmt.Errorf("Telegram deletion requires an interactive terminal")
	}
	chatID, _ := cmd.Flags().GetInt64("chat")
	revoke, _ := cmd.Flags().GetBool("revoke")
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	if err := requireSession(cfg); err != nil {
		return nil, err
	}
	db, err := openTelegramDB(cfg)
	if err != nil {
		return nil, err
	}
	chat, err := tgsrc.GetChat(db, chatID)
	if err == nil && chat == nil {
		err = fmt.Errorf("unknown chat %d; run 'obk telegram chats list' to find a synced chat", chatID)
	}
	var peer tg.InputPeerClass
	if err == nil {
		peer, err = tgsrc.InputPeerFor(chat)
	}
	if err != nil {
		db.Close()
		return nil, err
	}
	kind, rawID := tgsrc.SplitChatID(chatID)
	return &deleteTarget{cfg: cfg, db: db, chat: chat, peer: peer, kind: kind, rawID: rawID, revoke: revoke}, nil
}

func (t *deleteTarget) run(cmd *cobra.Command, fn func(context.Context, *tg.Client) error) error {
	confirmation := fmt.Sprintf("delete %d", t.chat.ChatID)
	fmt.Fprintf(cmd.OutOrStdout(), "Type %q to confirm: ", confirmation)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil || strings.TrimSpace(line) != confirmation {
		return fmt.Errorf("deletion cancelled")
	}
	client, err := newClient(t.cfg, t.db)
	if err != nil {
		return err
	}
	return client.Run(cmd.Context(), func(ctx context.Context) error { return fn(ctx, client.API()) })
}

func (t *deleteTarget) deleteRemoteChat(ctx context.Context, api *tg.Client) error {
	switch t.kind {
	case tgsrc.PeerChannel:
		p := t.peer.(*tg.InputPeerChannel)
		if _, err := api.ChannelsLeaveChannel(ctx, &tg.InputChannel{ChannelID: p.ChannelID, AccessHash: p.AccessHash}); err != nil {
			return fmt.Errorf("leave channel: %w", err)
		}
		return nil
	case tgsrc.PeerGroup:
		if _, err := api.MessagesDeleteChatUser(ctx, &tg.MessagesDeleteChatUserRequest{ChatID: t.rawID, UserID: &tg.InputUserSelf{}}); err != nil {
			return fmt.Errorf("leave group: %w", err)
		}
	}
	for {
		result, err := api.MessagesDeleteHistory(ctx, &tg.MessagesDeleteHistoryRequest{Peer: t.peer, Revoke: t.revoke})
		if err != nil {
			return fmt.Errorf("delete history: %w", err)
		}
		if result.Offset == 0 {
			return nil
		}
	}
}

func (t *deleteTarget) deleteRemoteMessages(ctx context.Context, api *tg.Client, ids []int) error {
	if p, ok := t.peer.(*tg.InputPeerChannel); ok {
		_, err := api.ChannelsDeleteMessages(ctx, &tg.ChannelsDeleteMessagesRequest{
			Channel: &tg.InputChannel{ChannelID: p.ChannelID, AccessHash: p.AccessHash}, ID: ids,
		})
		if err != nil {
			return fmt.Errorf("delete channel messages: %w", err)
		}
		return nil
	}
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
		message, ok := raw.AsNotEmpty()
		if !ok {
			continue
		}
		if id, ok := tgsrc.ChatIDFromPeer(message.GetPeerID()); !ok || id != t.chat.ChatID {
			return fmt.Errorf("message %d does not belong to chat %d", message.GetID(), t.chat.ChatID)
		}
		verified[message.GetID()] = true
	}
	for _, id := range ids {
		if !verified[id] {
			return fmt.Errorf("message %d was not found in chat %d", id, t.chat.ChatID)
		}
	}
	if _, err := api.MessagesDeleteMessages(ctx, &tg.MessagesDeleteMessagesRequest{ID: ids, Revoke: t.revoke}); err != nil {
		return fmt.Errorf("delete messages: %w", err)
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
