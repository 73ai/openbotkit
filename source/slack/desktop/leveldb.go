package desktop

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

var tokenRe = regexp.MustCompile(`xoxc-[a-zA-Z0-9-]+`)

func slackLevelDBPaths() []string {
	if runtime.GOOS != "darwin" {
		return nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{
		filepath.Join(home, "Library", "Application Support", "Slack", "Local Storage", "leveldb"),
		filepath.Join(home, "Library", "Containers", "com.tinyspeck.slackmacgap", "Data", "Library", "Application Support", "Slack", "Local Storage", "leveldb"),
	}
}

func ExtractToken() (string, error) {
	paths := slackLevelDBPaths()
	if len(paths) == 0 {
		return "", fmt.Errorf("slack desktop not supported on %s", runtime.GOOS)
	}

	for _, path := range paths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}
		token, err := extractTokenFromDB(path)
		if err != nil {
			continue
		}
		if token != "" {
			return token, nil
		}
	}
	return "", fmt.Errorf("no xoxc token found in Slack Desktop storage")
}

func extractTokenFromDB(path string) (string, error) {
	// Copy LevelDB files to a temp dir so we can open even when Slack Desktop
	// holds the LOCK file on the original database.
	tmpDir, err := copyLevelDB(path)
	if err != nil {
		return "", fmt.Errorf("copy leveldb: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	db, err := leveldb.OpenFile(tmpDir, &opt.Options{ReadOnly: true})
	if err != nil {
		return "", fmt.Errorf("open leveldb: %w", err)
	}
	defer db.Close()

	iter := db.NewIterator(nil, nil)
	defer iter.Release()

	for iter.Next() {
		val := string(iter.Value())
		if strings.Contains(val, "xoxc-") {
			if match := tokenRe.FindString(val); match != "" {
				return match, nil
			}
		}
	}
	return "", nil
}

// copyLevelDB copies .ldb, .log, and CURRENT files to a temp directory,
// bypassing the LOCK held by the running Slack Desktop process.
func copyLevelDB(src string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "obk-leveldb-*")
	if err != nil {
		return "", err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", err
	}

	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".ldb") && !strings.HasSuffix(name, ".log") && name != "CURRENT" && name != "MANIFEST-000001" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(src, name))
		if err != nil {
			continue
		}
		if err := os.WriteFile(filepath.Join(tmpDir, name), data, 0600); err != nil {
			os.RemoveAll(tmpDir)
			return "", err
		}
	}

	// Copy any MANIFEST file referenced by CURRENT.
	currentData, err := os.ReadFile(filepath.Join(src, "CURRENT"))
	if err == nil {
		manifest := strings.TrimSpace(string(currentData))
		if manifest != "MANIFEST-000001" {
			data, err := os.ReadFile(filepath.Join(src, manifest))
			if err == nil {
				os.WriteFile(filepath.Join(tmpDir, manifest), data, 0600)
			}
		}
	}

	return tmpDir, nil
}
