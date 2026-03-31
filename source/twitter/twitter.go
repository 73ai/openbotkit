package twitter

import (
	"context"

	"github.com/73ai/openbotkit/source"
	tclient "github.com/73ai/openbotkit/source/twitter/client"
	"github.com/73ai/openbotkit/store"
)

type X struct {
	cfg Config
}

func New(cfg Config) *X {
	return &X{cfg: cfg}
}

func (x *X) Name() string {
	return "x"
}

func (x *X) Status(ctx context.Context, db *store.DB) (*source.Status, error) {
	_, err := tclient.LoadSession()
	hasCredentials := err == nil

	count, _ := CountTweets(db)
	lastSync, _ := LastSyncTime(db)

	return &source.Status{
		Connected:    hasCredentials,
		ItemCount:    count,
		LastSyncedAt: lastSync,
	}, nil
}

func (x *X) Sync(ctx context.Context, db *store.DB, fetcher TimelineFetcher, opts SyncOptions) (*SyncResult, error) {
	return Sync(ctx, db, fetcher, opts)
}
