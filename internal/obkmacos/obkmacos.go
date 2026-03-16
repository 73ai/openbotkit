package obkmacos

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Contact struct {
	FirstName string   `json:"firstName"`
	LastName  string   `json:"lastName"`
	Nickname  string   `json:"nickname"`
	Phones    []string `json:"phones"`
	Emails    []string `json:"emails"`
}

type contactsResponse struct {
	Contacts []Contact `json:"contacts"`
}

type Note struct {
	ID                string `json:"id"`
	Title             string `json:"title"`
	Body              string `json:"body"`
	PasswordProtected bool   `json:"passwordProtected"`
	CreatedAt         string `json:"createdAt"`
	ModifiedAt        string `json:"modifiedAt"`
}

type Folder struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	ParentID string   `json:"parentId"`
	Account  string   `json:"account"`
	NoteIDs  []string `json:"noteIds"`
}

type NotesResponse struct {
	Notes   []Note   `json:"notes"`
	Folders []Folder `json:"folders"`
}

type PermissionStatus struct {
	Contacts string `json:"contacts"`
	Notes    string `json:"notes"`
}

type errorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func BinaryPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".obk", "bin", "obkmacos")
}

func Available() bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	_, err := os.Stat(BinaryPath())
	return err == nil
}

func FetchContacts() ([]Contact, error) {
	out, err := run("contacts")
	if err != nil {
		return nil, err
	}
	var resp contactsResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("obkmacos: parse contacts: %w", err)
	}
	return resp.Contacts, nil
}

func FetchNotes() (*NotesResponse, error) {
	out, err := run("notes")
	if err != nil {
		return nil, err
	}
	var resp NotesResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("obkmacos: parse notes: %w", err)
	}
	return &resp, nil
}

func CheckPermissions() (*PermissionStatus, error) {
	out, err := run("check")
	if err != nil {
		return nil, err
	}
	var status PermissionStatus
	if err := json.Unmarshal(out, &status); err != nil {
		return nil, fmt.Errorf("obkmacos: parse permissions: %w", err)
	}
	return &status, nil
}

func ParseNoteTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

func run(subcommand string) ([]byte, error) {
	binPath := BinaryPath()
	if _, err := os.Stat(binPath); err != nil {
		return nil, fmt.Errorf("obkmacos binary not found at %s — run 'make install' or 'obk setup'", binPath)
	}
	cmd := exec.Command(binPath, subcommand)
	out, err := cmd.CombinedOutput()
	if err != nil {
		var errResp errorResponse
		if jsonErr := json.Unmarshal(out, &errResp); jsonErr == nil && errResp.Error != "" {
			return nil, fmt.Errorf("obkmacos %s: %s", subcommand, errResp.Error)
		}
		return nil, fmt.Errorf("obkmacos %s: %s: %w", subcommand, strings.TrimSpace(string(out)), err)
	}
	return out, nil
}
