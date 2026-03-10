package imessage

import (
	"context"

	"github.com/priyanshujain/openbotkit/source"
	"github.com/priyanshujain/openbotkit/store"
)

type IMessage struct {
	cfg Config
}

func New(cfg Config) *IMessage {
	return &IMessage{cfg: cfg}
}

func (im *IMessage) Name() string {
	return "imessage"
}

func (im *IMessage) Status(ctx context.Context, db *store.DB) (*source.Status, error) {
	if db == nil {
		return &source.Status{Connected: false}, nil
	}

	count, _ := CountMessages(db)
	lastSync, _ := LastSyncTime(db)

	return &source.Status{
		Connected:    count > 0,
		ItemCount:    count,
		LastSyncedAt: lastSync,
	}, nil
}
