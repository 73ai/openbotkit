package channel

import (
	"context"
	"fmt"

	"github.com/73ai/openbotkit/config"
	wasrc "github.com/73ai/openbotkit/source/whatsapp"
	"github.com/73ai/openbotkit/store"
)

type WhatsAppPusher struct {
	client   *wasrc.Client
	account  string
	ownerJID string
	dataDB   *store.DB
}

var _ Pusher = (*WhatsAppPusher)(nil)

func NewWhatsAppPusher(cfg *config.Config, accountLabel, ownerJID string) (*WhatsAppPusher, error) {
	sessionDBPath := cfg.WhatsAppAccountSessionDBPath(accountLabel)
	client, err := wasrc.NewClient(context.Background(), sessionDBPath)
	if err != nil {
		return nil, fmt.Errorf("create whatsapp client: %w", err)
	}

	db, err := store.Open(store.Config{
		Driver: cfg.WhatsApp.Storage.Driver,
		DSN:    cfg.WhatsAppDataDSN(),
	})
	if err != nil {
		return nil, fmt.Errorf("open whatsapp data db: %w", err)
	}

	return &WhatsAppPusher{client: client, account: accountLabel, ownerJID: ownerJID, dataDB: db}, nil
}

func (p *WhatsAppPusher) Push(ctx context.Context, message string) error {
	_, err := wasrc.SendText(ctx, p.client, p.dataDB, wasrc.SendInput{
		Account: p.account,
		ChatJID: p.ownerJID,
		Text:    message,
	})
	return err
}
