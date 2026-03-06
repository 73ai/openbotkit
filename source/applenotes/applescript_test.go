package applenotes

import "testing"

func TestParseAppleScriptDate(t *testing.T) {
	tests := []struct {
		input string
		year  int
		month int
		day   int
	}{
		{"Monday, January 6, 2025 at 3:04:05 PM", 2025, 1, 6},
		{"January 6, 2025 at 3:04:05 PM", 2025, 1, 6},
		{"2025-01-06 15:04:05 -0700", 2025, 1, 6},
		{"2025-01-06T15:04:05Z", 2025, 1, 6},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseAppleScriptDate(tt.input)
			if got.IsZero() {
				t.Fatalf("failed to parse %q", tt.input)
			}
			if got.Year() != tt.year || int(got.Month()) != tt.month || got.Day() != tt.day {
				t.Errorf("got %v, want %d-%02d-%02d", got, tt.year, tt.month, tt.day)
			}
		})
	}
}

func TestParseAppleScriptDateInvalid(t *testing.T) {
	got := parseAppleScriptDate("not a date")
	if !got.IsZero() {
		t.Errorf("expected zero time for invalid input, got %v", got)
	}
}

func TestIsRecentlyDeletedFolder(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"Recently Deleted", true},
		{"recently deleted", true},
		{"Notes", false},
		{"Work", false},
		{"Récemment supprimées", true},
		{"Zuletzt gelöscht", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRecentlyDeletedFolder(tt.name)
			if got != tt.want {
				t.Errorf("isRecentlyDeletedFolder(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
