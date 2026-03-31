package client

import (
	"encoding/base64"
	"testing"
)

func TestGenerateTransactionID_NonEmpty(t *testing.T) {
	id := GenerateTransactionID("GET", "/graphql/test")
	if id == "" {
		t.Fatal("transaction ID should not be empty")
	}
}

func TestGenerateTransactionID_ValidBase64(t *testing.T) {
	id := GenerateTransactionID("GET", "/graphql/test")
	_, err := base64.StdEncoding.DecodeString(id)
	if err != nil {
		t.Fatalf("transaction ID is not valid base64: %v", err)
	}
}

func TestGenerateTransactionID_Unique(t *testing.T) {
	id1 := GenerateTransactionID("GET", "/graphql/test")
	id2 := GenerateTransactionID("GET", "/graphql/test")
	if id1 == id2 {
		t.Error("two transaction IDs should differ")
	}
}
