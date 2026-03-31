package cookies

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func ExtractFirefoxCookie(hosts []string, names []string) (map[string]string, error) {
	profiles := findFirefoxProfiles()
	if len(profiles) == 0 {
		return nil, fmt.Errorf("no Firefox profiles found")
	}

	var lastErr error
	for _, profile := range profiles {
		dbPath := filepath.Join(profile, "cookies.sqlite")
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			continue
		}

		result, err := extractFirefoxCookiesFromDB(dbPath, hosts, names)
		if err != nil {
			lastErr = err
			continue
		}
		if len(result) > 0 {
			return result, nil
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("firefox cookie extraction failed: %w", lastErr)
	}
	return nil, fmt.Errorf("no matching cookies found in Firefox")
}

func extractFirefoxCookiesFromDB(dbPath string, hosts, names []string) (map[string]string, error) {
	tmp, err := copyToTemp(dbPath, "firefox-cookies-*.db")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmp)

	db, err := sql.Open("sqlite", tmp+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("open firefox cookie db: %w", err)
	}
	defer db.Close()

	hostPlaceholders := make([]string, len(hosts))
	hostArgs := make([]any, len(hosts))
	for i, h := range hosts {
		hostPlaceholders[i] = "?"
		hostArgs[i] = h
	}

	namePlaceholders := make([]string, len(names))
	nameArgs := make([]any, len(names))
	for i, n := range names {
		namePlaceholders[i] = "?"
		nameArgs[i] = n
	}

	query := fmt.Sprintf(
		`SELECT name, value FROM moz_cookies WHERE host IN (%s) AND name IN (%s) ORDER BY expiry DESC`,
		strings.Join(hostPlaceholders, ","),
		strings.Join(namePlaceholders, ","),
	)

	args := append(hostArgs, nameArgs...)
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query firefox cookies: %w", err)
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			continue
		}
		if _, exists := result[name]; exists {
			continue
		}
		if value != "" {
			result[name] = value
		}
	}

	return result, nil
}

func findFirefoxProfiles() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	var baseDir string
	switch runtime.GOOS {
	case "darwin":
		baseDir = filepath.Join(home, "Library", "Application Support", "Firefox", "Profiles")
	case "linux":
		baseDir = filepath.Join(home, ".mozilla", "firefox")
	default:
		return nil
	}

	matches, err := filepath.Glob(filepath.Join(baseDir, "*.default*"))
	if err != nil {
		return nil
	}
	return matches
}
