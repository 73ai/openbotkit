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
	db, err := leveldb.OpenFile(path, &opt.Options{ReadOnly: true})
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
