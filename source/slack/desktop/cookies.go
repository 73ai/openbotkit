package desktop

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/pbkdf2"
)

const (
	slackKeychainService = "Slack Safe Storage"
	slackKeychainAccount = "Slack"
	pbkdf2Iterations     = 1003
	pbkdf2KeyLen         = 16
)

func slackCookiePaths() []string {
	if runtime.GOOS != "darwin" {
		return nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{
		filepath.Join(home, "Library", "Application Support", "Slack", "Cookies"),
		filepath.Join(home, "Library", "Containers", "com.tinyspeck.slackmacgap", "Data", "Library", "Application Support", "Slack", "Cookies"),
	}
}

func ExtractCookie() (string, error) {
	paths := slackCookiePaths()
	if len(paths) == 0 {
		return "", fmt.Errorf("cookie extraction not supported on %s", runtime.GOOS)
	}

	encKey, err := getKeychainPassword()
	if err != nil {
		return "", fmt.Errorf("get keychain password: %w", err)
	}

	derivedKey := pbkdf2.Key([]byte(encKey), []byte("saltysalt"), pbkdf2Iterations, pbkdf2KeyLen, sha1.New)

	for _, path := range paths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}
		cookie, err := extractCookieFromDB(path, derivedKey)
		if err != nil {
			continue
		}
		if cookie != "" {
			return cookie, nil
		}
	}
	return "", fmt.Errorf("no Slack 'd' cookie found")
}

func extractCookieFromDB(dbPath string, key []byte) (string, error) {
	// Copy to temp file since Slack may have the DB locked.
	tmp, err := copyToTemp(dbPath)
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp)

	db, err := sql.Open("sqlite3", tmp+"?mode=ro")
	if err != nil {
		return "", fmt.Errorf("open cookie db: %w", err)
	}
	defer db.Close()

	var encValue []byte
	err = db.QueryRow(
		`SELECT encrypted_value FROM cookies WHERE host_key = '.slack.com' AND name = 'd' ORDER BY expires_utc DESC LIMIT 1`,
	).Scan(&encValue)
	if err != nil {
		return "", fmt.Errorf("query cookie: %w", err)
	}

	return decryptCookie(encValue, key)
}

func decryptCookie(encrypted, key []byte) (string, error) {
	// Chromium on macOS prepends "v10" to the encrypted value.
	if len(encrypted) < 3 {
		return "", fmt.Errorf("encrypted value too short")
	}
	if string(encrypted[:3]) == "v10" {
		encrypted = encrypted[3:]
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	if len(encrypted) < aes.BlockSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	// Chromium uses 16 bytes of space as IV.
	iv := make([]byte, aes.BlockSize)
	for i := range iv {
		iv[i] = ' '
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	plaintext := make([]byte, len(encrypted))
	mode.CryptBlocks(plaintext, encrypted)

	// Remove PKCS7 padding.
	if len(plaintext) > 0 {
		padLen := int(plaintext[len(plaintext)-1])
		if padLen > 0 && padLen <= aes.BlockSize && padLen <= len(plaintext) {
			plaintext = plaintext[:len(plaintext)-padLen]
		}
	}

	return string(plaintext), nil
}

func getKeychainPassword() (string, error) {
	if runtime.GOOS != "darwin" {
		return "", fmt.Errorf("keychain access only supported on macOS")
	}
	out, err := exec.Command("security", "find-generic-password",
		"-s", slackKeychainService,
		"-a", slackKeychainAccount,
		"-w",
	).Output()
	if err != nil {
		return "", fmt.Errorf("security command: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func copyToTemp(src string) (string, error) {
	data, err := os.ReadFile(src)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", src, err)
	}
	f, err := os.CreateTemp("", "slack-cookies-*.db")
	if err != nil {
		return "", fmt.Errorf("create temp: %w", err)
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", fmt.Errorf("write temp: %w", err)
	}
	f.Close()
	return f.Name(), nil
}
