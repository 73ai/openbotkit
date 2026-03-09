package history

import (
	"context"

	"github.com/priyanshujain/openbotkit/source"
	"github.com/priyanshujain/openbotkit/store"
)

type History struct {
	cfg Config
}

func New(cfg Config) *History {
	return &History{cfg: cfg}
}

func (h *History) Name() string {
	return "history"
}

func (h *History) Status(ctx context.Context, db *store.DB) (*source.Status, error) {
	if db == nil {
		return &source.Status{Connected: false}, nil
	}

	count, _ := CountConversations(db)
	lastCapture, _ := LastCaptureTime(db)

	return &source.Status{
		Connected:    count > 0,
		ItemCount:    count,
		LastSyncedAt: lastCapture,
	}, nil
}
