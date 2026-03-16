package obkmacos

import (
	"encoding/json"
	"testing"
	"time"
)

func TestContactsParsing(t *testing.T) {
	raw := `{
		"contacts": [
			{"firstName": "Jane", "lastName": "Doe", "nickname": "JD", "phones": ["+1234567890"], "emails": ["jane@example.com"]},
			{"firstName": "Bob", "lastName": "", "nickname": "", "phones": [], "emails": ["bob@test.com"]}
		]
	}`
	var resp contactsResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(resp.Contacts) != 2 {
		t.Fatalf("expected 2 contacts, got %d", len(resp.Contacts))
	}
	c := resp.Contacts[0]
	if c.FirstName != "Jane" || c.LastName != "Doe" || c.Nickname != "JD" {
		t.Errorf("name fields: got %q %q %q", c.FirstName, c.LastName, c.Nickname)
	}
	if len(c.Phones) != 1 || c.Phones[0] != "+1234567890" {
		t.Errorf("phones: got %v", c.Phones)
	}
	if len(c.Emails) != 1 || c.Emails[0] != "jane@example.com" {
		t.Errorf("emails: got %v", c.Emails)
	}
}

func TestNotesParsing(t *testing.T) {
	raw := `{
		"notes": [
			{"id": "x-coredata://123", "title": "My Note", "body": "hello world", "passwordProtected": false, "createdAt": "2026-01-15T10:30:00Z", "modifiedAt": "2026-03-10T14:22:00Z"}
		],
		"folders": [
			{"id": "x-coredata://456", "name": "Work", "parentId": "", "account": "iCloud", "noteIds": ["x-coredata://123"]}
		]
	}`
	var resp NotesResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(resp.Notes) != 1 {
		t.Fatalf("expected 1 note, got %d", len(resp.Notes))
	}
	n := resp.Notes[0]
	if n.ID != "x-coredata://123" || n.Title != "My Note" {
		t.Errorf("note fields: got id=%q title=%q", n.ID, n.Title)
	}
	if n.PasswordProtected {
		t.Error("expected passwordProtected=false")
	}
	if len(resp.Folders) != 1 {
		t.Fatalf("expected 1 folder, got %d", len(resp.Folders))
	}
	f := resp.Folders[0]
	if f.Name != "Work" || f.Account != "iCloud" {
		t.Errorf("folder: got name=%q account=%q", f.Name, f.Account)
	}
	if len(f.NoteIDs) != 1 || f.NoteIDs[0] != "x-coredata://123" {
		t.Errorf("folder noteIds: got %v", f.NoteIDs)
	}
}

func TestPermissionsParsing(t *testing.T) {
	raw := `{"contacts": "authorized", "notes": "denied"}`
	var status PermissionStatus
	if err := json.Unmarshal([]byte(raw), &status); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if status.Contacts != "authorized" {
		t.Errorf("contacts: got %q", status.Contacts)
	}
	if status.Notes != "denied" {
		t.Errorf("notes: got %q", status.Notes)
	}
}

func TestErrorParsing(t *testing.T) {
	raw := `{"error": "Contacts access denied.", "code": "permission_denied"}`
	var resp errorResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if resp.Error != "Contacts access denied." {
		t.Errorf("error: got %q", resp.Error)
	}
	if resp.Code != "permission_denied" {
		t.Errorf("code: got %q", resp.Code)
	}
}

func TestParseNoteTime(t *testing.T) {
	tests := []struct {
		input string
		want  time.Time
	}{
		{"2026-01-15T10:30:00Z", time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)},
		{"", time.Time{}},
		{"not-a-date", time.Time{}},
	}
	for _, tt := range tests {
		got := ParseNoteTime(tt.input)
		if !got.Equal(tt.want) {
			t.Errorf("ParseNoteTime(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestContactsEmptyList(t *testing.T) {
	raw := `{"contacts": []}`
	var resp contactsResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(resp.Contacts) != 0 {
		t.Fatalf("expected 0 contacts, got %d", len(resp.Contacts))
	}
}
