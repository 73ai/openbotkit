package contacts

import (
	"testing"

	"github.com/73ai/openbotkit/internal/obkmacos"
)

func TestConvertContacts(t *testing.T) {
	input := []obkmacos.Contact{
		{FirstName: "Jane", LastName: "Doe", Nickname: "JD", Phones: []string{"+1234567890"}, Emails: []string{"jane@example.com"}},
		{FirstName: "Bob", LastName: "", Nickname: "", Phones: nil, Emails: []string{"bob@test.com"}},
	}
	got := convertContacts(input)
	if len(got) != 2 {
		t.Fatalf("expected 2 people, got %d", len(got))
	}
	if got[0].firstName != "Jane" || got[0].lastName != "Doe" || got[0].nickname != "JD" {
		t.Errorf("person[0] names: got %q %q %q", got[0].firstName, got[0].lastName, got[0].nickname)
	}
	if len(got[0].phones) != 1 || got[0].phones[0] != "+1234567890" {
		t.Errorf("person[0] phones: got %v", got[0].phones)
	}
	if len(got[1].emails) != 1 || got[1].emails[0] != "bob@test.com" {
		t.Errorf("person[1] emails: got %v", got[1].emails)
	}
}

func TestConvertContacts_FiltersEmpty(t *testing.T) {
	input := []obkmacos.Contact{
		{FirstName: "Jane", LastName: "Doe", Phones: []string{"+1234567890"}},
		{FirstName: "", LastName: "", Nickname: "", Phones: nil, Emails: nil},
		{FirstName: "", LastName: "", Nickname: "", Phones: []string{}, Emails: []string{}},
	}
	got := convertContacts(input)
	if len(got) != 1 {
		t.Fatalf("expected 1 person (2 empty filtered), got %d", len(got))
	}
	if got[0].firstName != "Jane" {
		t.Errorf("expected Jane, got %q", got[0].firstName)
	}
}

func TestConvertContacts_NicknameOnlyKept(t *testing.T) {
	input := []obkmacos.Contact{
		{FirstName: "", LastName: "", Nickname: "Ghost", Phones: nil, Emails: nil},
	}
	got := convertContacts(input)
	// Nickname alone doesn't prevent filtering (no name, no phones, no emails)
	if len(got) != 0 {
		t.Fatalf("expected 0 people (nickname-only filtered), got %d", len(got))
	}
}

func TestConvertContacts_Empty(t *testing.T) {
	got := convertContacts(nil)
	if len(got) != 0 {
		t.Fatalf("expected 0 people, got %d", len(got))
	}
}
