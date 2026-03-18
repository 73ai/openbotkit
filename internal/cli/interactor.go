package cli

import (
	"github.com/73ai/openbotkit/agent/tools"
	clicli "github.com/73ai/openbotkit/channel/cli"
)

// CLIInteractor adapts a CLI channel to the tools.Interactor interface.
type CLIInteractor struct {
	ch *clicli.Channel
}

var _ tools.Interactor = (*CLIInteractor)(nil)

// NewCLIInteractor creates an interactor backed by the given CLI channel.
func NewCLIInteractor(ch *clicli.Channel) *CLIInteractor {
	return &CLIInteractor{ch: ch}
}

func (c *CLIInteractor) Notify(msg string) error                   { return c.ch.Send(msg) }
func (c *CLIInteractor) NotifyLink(text, url string) error         { return c.ch.SendLink(text, url) }
func (c *CLIInteractor) RequestApproval(desc string) (bool, error) { return c.ch.RequestApproval(desc) }
