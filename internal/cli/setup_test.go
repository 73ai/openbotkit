package cli

import "testing"

func TestCleanPath(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"  /path/to/file  ", "/path/to/file"},
		{`"/path/to/file"`, "/path/to/file"},
		{`'/path/to/file'`, "/path/to/file"},
		{`  "/path/to/file"  `, "/path/to/file"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := cleanPath(tt.in); got != tt.want {
			t.Errorf("cleanPath(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestIsGWSService(t *testing.T) {
	for _, svc := range []string{"calendar", "drive", "docs", "sheets", "tasks", "people"} {
		if !isGWSService(svc) {
			t.Errorf("isGWSService(%q) = false, want true", svc)
		}
	}
	if isGWSService("gmail") {
		t.Error("isGWSService(gmail) = true, want false")
	}
	if isGWSService("unknown") {
		t.Error("isGWSService(unknown) = true, want false")
	}
}
