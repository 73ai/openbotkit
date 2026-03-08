package daemon

import (
	"path/filepath"

	"github.com/priyanshujain/openbotkit/config"
)

func lockPath() string {
	return filepath.Join(config.Dir(), "daemon.lock")
}
