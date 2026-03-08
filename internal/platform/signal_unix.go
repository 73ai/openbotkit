//go:build !windows

package platform

import (
	"os"
	"syscall"
)

var ShutdownSignals = []os.Signal{os.Interrupt, syscall.SIGTERM}
