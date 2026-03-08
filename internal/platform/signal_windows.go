//go:build windows

package platform

import "os"

var ShutdownSignals = []os.Signal{os.Interrupt}
