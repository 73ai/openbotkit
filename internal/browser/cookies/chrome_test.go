package cookies

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"database/sql"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/pbkdf2"
)

func TestDecryptChromeValue_RoundTrip(t *testing.T) {
	password := "test-password"
	key := pbkdf2.Key([]byte(password), []byte("saltysalt"), chromePBKDF2Iterations, chromePBKDF2KeyLen, sha1.New)

	plaintext := "my-secret-cookie-value"
	encrypted := encryptForTest(t, []byte(plaintext), key)

	got, err := DecryptChromeValue(encrypted, key)
	if err != nil {
		t.Fatalf("DecryptChromeValue: %v", err)
	}
	if got != plaintext {
		t.Errorf("got %q, want %q", got, plaintext)
	}
}

func TestDecryptChromeValue_BadPrefix(t *testing.T) {
	key := make([]byte, 16)
	encrypted := []byte("v11" + "some-encrypted-data!")

	_, err := DecryptChromeValue(encrypted, key)
	if err == nil {
		t.Fatal("expected error for bad version prefix")
	}
	if got := err.Error(); got != `unsupported encryption version "v11"` {
		t.Errorf("unexpected error: %s", got)
	}
}

func TestDecryptChromeValue_TooShort(t *testing.T) {
	key := make([]byte, 16)

	_, err := DecryptChromeValue([]byte("v1"), key)
	if err == nil {
		t.Fatal("expected error for too-short input")
	}
}

func TestDecryptChromeValue_BadCiphertextLength(t *testing.T) {
	key := make([]byte, 16)
	// v10 prefix + 5 bytes (not a multiple of block size)
	encrypted := append([]byte("v10"), make([]byte, 5)...)

	_, err := DecryptChromeValue(encrypted, key)
	if err == nil {
		t.Fatal("expected error for bad ciphertext length")
	}
}

func TestExtractChromeCookiesFromDB(t *testing.T) {
	password := "test-password"
	key := pbkdf2.Key([]byte(password), []byte("saltysalt"), chromePBKDF2Iterations, chromePBKDF2KeyLen, sha1.New)

	dbPath := createChromeCookieDB(t, key, []chromeCookieRow{
		{host: ".x.com", name: "auth_token", value: "abc123secret"},
		{host: ".x.com", name: "ct0", value: "csrf456token"},
		{host: ".google.com", name: "other", value: "irrelevant"},
	})

	hosts := []string{".x.com", "x.com", ".twitter.com", "twitter.com"}
	names := []string{"auth_token", "ct0"}

	result, err := extractChromeCookiesFromDB(dbPath, key, hosts, names)
	if err != nil {
		t.Fatalf("extractChromeCookiesFromDB: %v", err)
	}

	if got := result["auth_token"]; got != "abc123secret" {
		t.Errorf("auth_token = %q, want %q", got, "abc123secret")
	}
	if got := result["ct0"]; got != "csrf456token" {
		t.Errorf("ct0 = %q, want %q", got, "csrf456token")
	}
}

func TestExtractChromeCookiesFromDB_Empty(t *testing.T) {
	key := make([]byte, 16)
	dbPath := createChromeCookieDB(t, key, nil)

	hosts := []string{".x.com"}
	names := []string{"auth_token"}

	result, err := extractChromeCookiesFromDB(dbPath, key, hosts, names)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result, got %v", result)
	}
}

func TestExtractChromeCookiesFromDB_MultipleHosts(t *testing.T) {
	password := "test-password"
	key := pbkdf2.Key([]byte(password), []byte("saltysalt"), chromePBKDF2Iterations, chromePBKDF2KeyLen, sha1.New)

	dbPath := createChromeCookieDB(t, key, []chromeCookieRow{
		{host: ".twitter.com", name: "auth_token", value: "twitter-auth"},
		{host: ".x.com", name: "ct0", value: "x-csrf"},
	})

	hosts := []string{".x.com", "x.com", ".twitter.com", "twitter.com"}
	names := []string{"auth_token", "ct0"}

	result, err := extractChromeCookiesFromDB(dbPath, key, hosts, names)
	if err != nil {
		t.Fatalf("extractChromeCookiesFromDB: %v", err)
	}

	if got := result["auth_token"]; got != "twitter-auth" {
		t.Errorf("auth_token = %q, want %q", got, "twitter-auth")
	}
	if got := result["ct0"]; got != "x-csrf" {
		t.Errorf("ct0 = %q, want %q", got, "x-csrf")
	}
}

// --- helpers ---

type chromeCookieRow struct {
	host  string
	name  string
	value string
}

func createChromeCookieDB(t *testing.T, key []byte, rows []chromeCookieRow) string {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "Cookies")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE cookies (
		host_key TEXT NOT NULL,
		name TEXT NOT NULL,
		encrypted_value BLOB,
		expires_utc INTEGER DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	for _, r := range rows {
		enc := encryptForTest(t, []byte(r.value), key)
		_, err := db.Exec(`INSERT INTO cookies (host_key, name, encrypted_value, expires_utc) VALUES (?, ?, ?, ?)`,
			r.host, r.name, enc, 13300000000000000)
		if err != nil {
			t.Fatalf("insert row: %v", err)
		}
	}

	return dbPath
}

func encryptForTest(t *testing.T, plaintext, key []byte) []byte {
	t.Helper()

	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatalf("create cipher: %v", err)
	}

	// PKCS7 padding
	padLen := aes.BlockSize - len(plaintext)%aes.BlockSize
	padded := make([]byte, len(plaintext)+padLen)
	copy(padded, plaintext)
	for i := len(plaintext); i < len(padded); i++ {
		padded[i] = byte(padLen)
	}

	iv := make([]byte, aes.BlockSize)
	for i := range iv {
		iv[i] = ' '
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	ciphertext := make([]byte, len(padded))
	mode.CryptBlocks(ciphertext, padded)

	return append([]byte("v10"), ciphertext...)
}
