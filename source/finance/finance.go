package finance

import (
	"context"

	"github.com/priyanshujain/openbotkit/source"
	"github.com/priyanshujain/openbotkit/store"
)

type Finance struct {
	cfg Config
}

func New(cfg Config) *Finance {
	return &Finance{cfg: cfg}
}

func (f *Finance) Name() string {
	return "finance"
}

func (f *Finance) Status(_ context.Context, _ *store.DB) (*source.Status, error) {
	return &source.Status{Connected: true}, nil
}
