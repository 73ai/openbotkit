package memory

import "os"

// EnsureDir creates the memory store directory if it doesn't exist.
func EnsureDir(dir string) error {
	return os.MkdirAll(dir, 0700)
}
