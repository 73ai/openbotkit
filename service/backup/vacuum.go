package backup

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

func VacuumInto(dbPath, stagingDir string) (string, error) {
	rel := strings.TrimPrefix(dbPath, "/")
	destPath := filepath.Join(stagingDir, filepath.Base(rel))
	if err := os.MkdirAll(filepath.Dir(destPath), 0700); err != nil {
		return "", fmt.Errorf("create staging dir: %w", err)
	}

	os.Remove(destPath)

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", dbPath, err)
	}
	defer db.Close()

	if _, err := db.Exec(fmt.Sprintf("VACUUM INTO '%s'", destPath)); err != nil {
		return "", fmt.Errorf("vacuum into %s: %w", destPath, err)
	}

	return destPath, nil
}
