package backup

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// VacuumInto creates a consistent snapshot of a SQLite database.
// relPath is the path relative to ~/.obk (e.g. "gmail/data.db") used to
// preserve directory structure in the staging dir and avoid collisions.
func VacuumInto(dbPath, stagingDir, relPath string) (string, error) {
	destPath := filepath.Join(stagingDir, relPath)
	if err := os.MkdirAll(filepath.Dir(destPath), 0700); err != nil {
		return "", fmt.Errorf("create staging dir: %w", err)
	}

	os.Remove(destPath)

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", dbPath, err)
	}
	defer db.Close()

	escaped := strings.ReplaceAll(destPath, "'", "''")
	if _, err := db.Exec(fmt.Sprintf("VACUUM INTO '%s'", escaped)); err != nil {
		return "", fmt.Errorf("vacuum into %s: %w", destPath, err)
	}

	return destPath, nil
}
