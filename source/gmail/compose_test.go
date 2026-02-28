package gmail

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestComposeRawMessage(t *testing.T) {
	tests := []struct {
		name      string
		input     ComposeInput
		wantErr   bool
		wantParts []string // substrings that must appear in the decoded message
		noParts   []string // substrings that must NOT appear
	}{
		{
			name: "basic message",
			input: ComposeInput{
				To:      []string{"alice@example.com"},
				Subject: "Hello",
				Body:    "Hi there",
				Account: "me@example.com",
			},
			wantParts: []string{
				"From: me@example.com\r\n",
				"To: alice@example.com\r\n",
				"Subject: Hello\r\n",
				"MIME-Version: 1.0\r\n",
				"Content-Type: text/plain; charset=\"UTF-8\"\r\n",
				"\r\nHi there",
			},
		},
		{
			name: "multiple recipients",
			input: ComposeInput{
				To:      []string{"a@test.com", "b@test.com"},
				Subject: "Multi",
				Body:    "body",
				Account: "me@test.com",
			},
			wantParts: []string{
				"To: a@test.com, b@test.com\r\n",
			},
		},
		{
			name: "with cc and bcc",
			input: ComposeInput{
				To:      []string{"a@test.com"},
				Cc:      []string{"cc@test.com"},
				Bcc:     []string{"bcc@test.com"},
				Subject: "Test",
				Body:    "body",
				Account: "me@test.com",
			},
			wantParts: []string{
				"Cc: cc@test.com\r\n",
				"Bcc: bcc@test.com\r\n",
			},
		},
		{
			name: "no cc or bcc omits headers",
			input: ComposeInput{
				To:      []string{"a@test.com"},
				Subject: "Test",
				Body:    "body",
				Account: "me@test.com",
			},
			noParts: []string{"Cc:", "Bcc:"},
		},
		{
			name: "empty recipients returns error",
			input: ComposeInput{
				Subject: "Test",
				Body:    "body",
				Account: "me@test.com",
			},
			wantErr: true,
		},
		{
			name: "empty body is valid",
			input: ComposeInput{
				To:      []string{"a@test.com"},
				Subject: "No body",
				Account: "me@test.com",
			},
			wantParts: []string{
				"Subject: No body\r\n",
				"\r\n", // headers end with blank line
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := composeRawMessage(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			decoded, err := base64.URLEncoding.DecodeString(encoded)
			if err != nil {
				t.Fatalf("decode base64url: %v", err)
			}
			msg := string(decoded)

			for _, part := range tt.wantParts {
				if !strings.Contains(msg, part) {
					t.Errorf("missing expected part %q in:\n%s", part, msg)
				}
			}
			for _, part := range tt.noParts {
				if strings.Contains(msg, part) {
					t.Errorf("found unwanted part %q in:\n%s", part, msg)
				}
			}
		})
	}
}

func TestComposeRawMessageIsBase64URL(t *testing.T) {
	encoded, err := composeRawMessage(ComposeInput{
		To:      []string{"a@test.com"},
		Subject: "Test",
		Body:    "Hello world",
		Account: "me@test.com",
	})
	if err != nil {
		t.Fatal(err)
	}

	// base64url must not contain + or /
	if strings.ContainsAny(encoded, "+/") {
		t.Errorf("encoded contains non-URL-safe characters: %s", encoded)
	}

	// Must decode cleanly
	_, err = base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		t.Errorf("not valid base64url: %v", err)
	}
}
