package desktop

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/pbkdf2"
)

func TestDecryptCookie(t *testing.T) {
	password := "testpassword"
	key := pbkdf2.Key([]byte(password), []byte("saltysalt"), pbkdf2Iterations, pbkdf2KeyLen, sha1.New)

	plaintext := []byte("xoxd-test-cookie-value")

	// Pad to AES block size (PKCS7).
	padLen := aes.BlockSize - (len(plaintext) % aes.BlockSize)
	padded := make([]byte, len(plaintext)+padLen)
	copy(padded, plaintext)
	for i := len(plaintext); i < len(padded); i++ {
		padded[i] = byte(padLen)
	}

	// Encrypt with CBC using space IV.
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	iv := make([]byte, aes.BlockSize)
	for i := range iv {
		iv[i] = ' '
	}
	ciphertext := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ciphertext, padded)

	// Prepend "v10" header.
	encrypted := append([]byte("v10"), ciphertext...)

	got, err := decryptCookie(encrypted, key)
	if err != nil {
		t.Fatal(err)
	}
	if got != "xoxd-test-cookie-value" {
		t.Errorf("decrypted = %q", got)
	}
}

func TestDecryptCookie_TooShort(t *testing.T) {
	_, err := decryptCookie([]byte("ab"), nil)
	if err == nil {
		t.Fatal("expected error for short input")
	}
}

func TestExtractCookieFromDB(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "Cookies")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	db.Exec(`CREATE TABLE cookies (host_key TEXT, name TEXT, encrypted_value BLOB, expires_utc INTEGER)`)

	password := "testpassword"
	key := pbkdf2.Key([]byte(password), []byte("saltysalt"), pbkdf2Iterations, pbkdf2KeyLen, sha1.New)

	plaintext := []byte("xoxd-test-value")
	padLen := aes.BlockSize - (len(plaintext) % aes.BlockSize)
	padded := make([]byte, len(plaintext)+padLen)
	copy(padded, plaintext)
	for i := len(plaintext); i < len(padded); i++ {
		padded[i] = byte(padLen)
	}
	block, _ := aes.NewCipher(key)
	iv := make([]byte, aes.BlockSize)
	for i := range iv {
		iv[i] = ' '
	}
	ciphertext := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ciphertext, padded)
	encrypted := append([]byte("v10"), ciphertext...)

	db.Exec(`INSERT INTO cookies (host_key, name, encrypted_value, expires_utc) VALUES (?, ?, ?, ?)`,
		".slack.com", "d", encrypted, 999999999)
	db.Close()

	got, err := extractCookieFromDB(dbPath, key)
	if err != nil {
		t.Fatal(err)
	}
	if got != "xoxd-test-value" {
		t.Errorf("cookie = %q", got)
	}
}

func TestExtractCookieFromDB_MissingFile(t *testing.T) {
	key := make([]byte, 16)
	_, err := extractCookieFromDB("/nonexistent/Cookies", key)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestCopyToTemp(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.txt")
	os.WriteFile(src, []byte("hello"), 0644)

	tmp, err := copyToTemp(src)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp)

	data, _ := os.ReadFile(tmp)
	if string(data) != "hello" {
		t.Errorf("content = %q", string(data))
	}
}

func TestCopyToTemp_MissingFile(t *testing.T) {
	_, err := copyToTemp("/nonexistent/file")
	if err == nil {
		t.Fatal("expected error")
	}
}
