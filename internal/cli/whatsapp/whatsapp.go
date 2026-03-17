package whatsapp

import (
	"fmt"

	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/provider"
	"github.com/73ai/openbotkit/remote"
	"github.com/spf13/cobra"
)

func newRemoteClient(cfg *config.Config) (*remote.Client, error) {
	if cfg.Remote == nil || cfg.Remote.Server == "" {
		return nil, fmt.Errorf("remote server not configured — run 'obk setup' to configure")
	}
	pw, err := cfg.Remote.ResolvedPassword(provider.LoadCredential)
	if err != nil {
		return nil, fmt.Errorf("remote password: %w", err)
	}
	return remote.NewClient(cfg.Remote.Server, cfg.Remote.Username, pw), nil
}

var Cmd = &cobra.Command{
	Use:   "whatsapp",
	Short: "Manage WhatsApp data source",
}

func init() {
	Cmd.AddCommand(authCmd)
	Cmd.AddCommand(syncCmd)
	Cmd.AddCommand(messagesCmd)
	Cmd.AddCommand(chatsCmd)
	Cmd.AddCommand(contactsCmd)
}
