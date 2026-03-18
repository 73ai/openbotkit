package contacts

import "testing"

func TestNormalizePhone(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"+1 (555) 123-4567", "+15551234567"},
		{"919876543210", "+919876543210"},
		{"+44 7700 900000", "+447700900000"},
		{"", ""},
		{"abc", ""},
		{"+1-555-123-4567", "+15551234567"},
	}
	for _, tt := range tests {
		got := NormalizePhone(tt.input)
		if got != tt.want {
			t.Errorf("NormalizePhone(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestNormalizeEmail(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Alice@Example.COM", "alice@example.com"},
		{"  bob@test.com  ", "bob@test.com"},
		{"", ""},
	}
	for _, tt := range tests {
		got := NormalizeEmail(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeEmail(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseEmailAddr(t *testing.T) {
	tests := []struct {
		input     string
		wantName  string
		wantEmail string
	}{
		{"Alice Smith <alice@example.com>", "Alice Smith", "alice@example.com"},
		{"<bob@test.com>", "", "bob@test.com"},
		{"charlie@test.com", "", "charlie@test.com"},
		{"David Chen <David@Example.COM>", "David Chen", "david@example.com"},
		{"not-an-email", "", ""},
		{"", "", ""},
	}
	for _, tt := range tests {
		name, email := ParseEmailAddr(tt.input)
		if name != tt.wantName || email != tt.wantEmail {
			t.Errorf("ParseEmailAddr(%q) = (%q, %q), want (%q, %q)",
				tt.input, name, email, tt.wantName, tt.wantEmail)
		}
	}
}

func TestExtractPhoneFromJID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"919876543210@s.whatsapp.net", "+919876543210"},
		{"15551234567@s.whatsapp.net", "+15551234567"},
		{"120363001234@g.us", "+120363001234"},
		{"", ""},
		{"noatsign", ""},
		{"@s.whatsapp.net", ""},
	}
	for _, tt := range tests {
		got := ExtractPhoneFromJID(tt.input)
		if got != tt.want {
			t.Errorf("ExtractPhoneFromJID(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
