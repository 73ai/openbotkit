package slack

import (
	"context"

	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/source"
	"github.com/73ai/openbotkit/store"
)

type Slack struct {
	cfg *config.SlackConfig
}

type Config struct {
	Slack *config.SlackConfig
}

func New(cfg Config) *Slack {
	return &Slack{cfg: cfg.Slack}
}

func (s *Slack) Name() string { return "slack" }

func (s *Slack) Status(_ context.Context, _ *store.DB) (*source.Status, error) {
	if s.cfg == nil || len(s.cfg.Workspaces) == 0 {
		return &source.Status{Connected: false}, nil
	}

	var accounts []string
	connected := false
	for name := range s.cfg.Workspaces {
		if _, err := LoadCredentials(name); err == nil {
			connected = true
			accounts = append(accounts, name)
		}
	}

	return &source.Status{
		Connected: connected,
		Accounts:  accounts,
		ItemCount: 0,
	}, nil
}
