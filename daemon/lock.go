package daemon

import (
	"path/filepath"

	"github.com/73ai/openbotkit/config"
)

func lockPath() string {
	return filepath.Join(config.Dir(), "daemon.lock")
}
