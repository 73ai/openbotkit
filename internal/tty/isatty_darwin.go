//go:build darwin

package tty

import (
	"syscall"
	"unsafe"
)

func isTerminal(fd uintptr) bool {
	var t syscall.Termios
	_, _, err := syscall.Syscall6(syscall.SYS_IOCTL, fd, syscall.TIOCGETA,
		uintptr(unsafe.Pointer(&t)), 0, 0, 0)
	return err == 0
}
