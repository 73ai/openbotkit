package cli

import (
	"fmt"

	"github.com/priyanshujain/openbotkit/config"
	"github.com/priyanshujain/openbotkit/provider"
	"github.com/priyanshujain/openbotkit/remote"
)

func remoteClient(cfg *config.Config) (*remote.Client, error) {
	if cfg.Remote == nil || cfg.Remote.Server == "" {
		return nil, fmt.Errorf("remote server not configured — run 'obk setup' to configure")
	}
	pw, err := cfg.Remote.ResolvedPassword(provider.LoadCredential)
	if err != nil {
		return nil, fmt.Errorf("remote password: %w", err)
	}
	return remote.NewClient(cfg.Remote.Server, cfg.Remote.Username, pw), nil
}
