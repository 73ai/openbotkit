package applenotes

import "testing"

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
