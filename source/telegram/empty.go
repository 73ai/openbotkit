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
	targets := []backfillTarget{}
	if err := f.Dialogs(ctx, func(ctx context.Context, elem dialogs.Elem) error {
		chat, ok := chatFromDialog(elem)
		if !ok {
			return fmt.Errorf("dialog peer unavailable: %T", elem.Dialog)
		}
		if chat.Type == PeerUser {
			chat.IsCurrentDialog = true
			targets = append(targets, backfillTarget{peer: elem.Peer, chat: chat})
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("iterate dialogs: %w", err)
	}

	found := []EmptyChat{}
	for _, target := range targets {
		regular, joined, other := false, false, false
		err := f.History(ctx, target.peer, 0, func(ctx context.Context, elem messages.Elem) error {
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
			return nil, fmt.Errorf("verify chat %d: %w", target.chat.ChatID, err)
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
		found = append(found, EmptyChat{Chat: *target.chat, Kind: kind})
	}
	return found, nil
}
