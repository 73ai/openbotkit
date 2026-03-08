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

// lockHandle stores the Windows file handle obtained during acquireLock
// so that releaseLock can unlock the same handle. On Windows, os.File.Fd()
// returns a new duplicated handle on each call, so we must capture it once.
var lockHandle uintptr

// acquireLock takes an exclusive file lock to prevent multiple daemon instances.
// Returns the lock file which must be kept open for the lifetime of the daemon.
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

// releaseLock releases the file lock and removes the lock file.
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
