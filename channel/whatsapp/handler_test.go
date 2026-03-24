package whatsapp

import "testing"

func TestShouldHandle_AcceptsOwnerDM(t *testing.T) {
	if !shouldHandle("123@s.whatsapp.net", "123@s.whatsapp.net", false, false, "hello") {
		t.Fatal("expected true for owner DM")
	}
}

func TestShouldHandle_RejectsNonOwner(t *testing.T) {
	if shouldHandle("999@s.whatsapp.net", "123@s.whatsapp.net", false, false, "hello") {
		t.Fatal("expected false for non-owner")
	}
}

func TestShouldHandle_RejectsFromMe(t *testing.T) {
	if shouldHandle("123@s.whatsapp.net", "123@s.whatsapp.net", true, false, "hello") {
		t.Fatal("expected false for isFromMe")
	}
}

func TestShouldHandle_RejectsGroupMessages(t *testing.T) {
	if shouldHandle("123@s.whatsapp.net", "123@s.whatsapp.net", false, true, "hello") {
		t.Fatal("expected false for group")
	}
}

func TestShouldHandle_RejectsEmptyText(t *testing.T) {
	if shouldHandle("123@s.whatsapp.net", "123@s.whatsapp.net", false, false, "") {
		t.Fatal("expected false for empty text")
	}
	if shouldHandle("123@s.whatsapp.net", "123@s.whatsapp.net", false, false, "   ") {
		t.Fatal("expected false for whitespace-only text")
	}
}

func TestJidUser_ExtractsUserPart(t *testing.T) {
	tests := []struct {
		jid  string
		want string
	}{
		{"123@s.whatsapp.net", "123"},
		{"456", "456"},
		{"789@g.us", "789"},
	}
	for _, tt := range tests {
		got := jidUser(tt.jid)
		if got != tt.want {
			t.Errorf("jidUser(%q) = %q, want %q", tt.jid, got, tt.want)
		}
	}
}
