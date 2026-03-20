package history

import (
	"os"
	"path/filepath"
)

func EnsureDir(dir string) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	return os.MkdirAll(filepath.Join(dir, "sessions"), 0700)
}
