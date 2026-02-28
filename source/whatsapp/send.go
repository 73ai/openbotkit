package whatsapp

import (
	"context"
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"

	"github.com/priyanshujain/openbotkit/store"
)

func SendText(ctx context.Context, client *Client, db *store.DB, input SendInput) (*SendResult, error) {
	jid, err := types.ParseJID(input.ChatJID)
	if err != nil {
		return nil, fmt.Errorf("parse JID %q: %w", input.ChatJID, err)
	}

	if err := client.Connect(ctx); err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	resp, err := client.WM().SendMessage(ctx, jid, &waE2E.Message{
		Conversation: proto.String(input.Text),
	})
	if err != nil {
		return nil, fmt.Errorf("send message: %w", err)
	}

	isGroup := strings.HasSuffix(input.ChatJID, "@g.us")

	msg := &Message{
		MessageID: resp.ID,
		ChatJID:   input.ChatJID,
		SenderJID: client.WM().Store.ID.String(),
		Text:      input.Text,
		Timestamp: resp.Timestamp,
		IsGroup:   isGroup,
		IsFromMe:  true,
	}
	if err := SaveMessage(db, msg); err != nil {
		return nil, fmt.Errorf("save sent message: %w", err)
	}

	UpsertChat(db, input.ChatJID, "", isGroup)

	return &SendResult{
		MessageID: resp.ID,
		Timestamp: resp.Timestamp,
	}, nil
}
