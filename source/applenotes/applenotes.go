package applenotes

import (
	"context"

	"github.com/73ai/openbotkit/source"
	"github.com/73ai/openbotkit/store"
)

type AppleNotes struct {
	cfg Config
}

func New(cfg Config) *AppleNotes {
	return &AppleNotes{cfg: cfg}
}

func (a *AppleNotes) Name() string {
	return "applenotes"
}

func (a *AppleNotes) Status(ctx context.Context, db *store.DB) (*source.Status, error) {
	if db == nil {
		return &source.Status{Connected: false}, nil
	}

	count, _ := CountNotes(db)
	lastSync, _ := LastSyncTime(db)

	return &source.Status{
		Connected:    count > 0,
		ItemCount:    count,
		LastSyncedAt: lastSync,
	}, nil
}
