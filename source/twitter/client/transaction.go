package client

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// GenerateTransactionID returns a random base64 transaction ID.
// TODO(phase2): Implement full twikit SVG-parsing algorithm for strict validation.
func GenerateTransactionID(method, path string) string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand failed: %v", err))
	}
	return base64.StdEncoding.EncodeToString(b)
}
