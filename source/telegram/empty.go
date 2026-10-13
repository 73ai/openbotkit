package telegram

import (
	"context"
	"errors"
	"fmt"

	"github.com/gotd/td/telegram/query/dialogs"
	"github.com/gotd/td/telegram/query/messages"
	"github.com/gotd/td/tg"
)

type EmptyChat struct {
	Chat Chat
	Kind string
}

func FindEmptyChats(ctx context.Context, f Fetcher) ([]EmptyChat, error) {
	var chats []*Chat
	var peers []tg.InputPeerClass
	if err := f.Dialogs(ctx, func(ctx context.Context, elem dialogs.Elem) error {
		if chat, ok := chatFromDialog(elem); ok && chat.Type == PeerUser {
			chats = append(chats, chat)
			peers = append(peers, elem.Peer)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("iterate dialogs: %w", err)
	}

	found := []EmptyChat{}
	for i, chat := range chats {
		regular, joined, other := false, false, false
		err := f.History(ctx, peers[i], 0, func(ctx context.Context, elem messages.Elem) error {
			switch msg := elem.Msg.(type) {
			case *tg.Message:
				regular = true
				return errStopChat
			case *tg.MessageService:
				if _, ok := msg.Action.(*tg.MessageActionContactSignUp); ok {
					joined = true
				} else {
					other = true
				}
			default:
				return fmt.Errorf("unverifiable history message: %T", elem.Msg)
			}
			return nil
		})
		if err != nil && !errors.Is(err, errStopChat) {
			return nil, fmt.Errorf("verify chat %d: %w", chat.ChatID, err)
		}
		if regular {
			continue
		}
		kind := "empty"
		if other {
			kind = "service_only"
		} else if joined {
			kind = "joined_telegram"
		}
		found = append(found, EmptyChat{Chat: *chat, Kind: kind})
	}
	return found, nil
}
