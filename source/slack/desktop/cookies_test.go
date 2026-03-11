package desktop

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"testing"

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
