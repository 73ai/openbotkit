//go:build windows

package daemon

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

var (
	modkernel32      = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = modkernel32.NewProc("LockFileEx")
	procUnlockFileEx = modkernel32.NewProc("UnlockFileEx")
)

const (
	lockfileExclusiveLock   = 0x00000002
	lockfileFailImmediately = 0x00000001
)

// On Windows, os.File.Fd() returns a new duplicated handle on each call,
// so we capture it once during acquireLock for use in releaseLock.
var lockHandle uintptr

func acquireLock() (*os.File, error) {
	f, err := os.OpenFile(lockPath(), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}

	h := f.Fd()
	ol := new(syscall.Overlapped)
	r1, _, _ := procLockFileEx.Call(
		uintptr(h),
		uintptr(lockfileExclusiveLock|lockfileFailImmediately),
		0,
		1, 0,
		uintptr(unsafe.Pointer(ol)),
	)
	if r1 == 0 {
		f.Close()
		return nil, fmt.Errorf("daemon is already running")
	}

	lockHandle = h
	return f, nil
}

func releaseLock(f *os.File) {
	ol := new(syscall.Overlapped)
	procUnlockFileEx.Call(
		lockHandle,
		0,
		1, 0,
		uintptr(unsafe.Pointer(ol)),
	)
	f.Close()
	os.Remove(lockPath())
}
