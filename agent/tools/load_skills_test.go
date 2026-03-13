package tools

import (
	"strings"
	"testing"
)

func TestEnsureGWSShared(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{"gws skill auto-includes shared", []string{"gws-drive"}, []string{"gws-shared", "gws-drive"}},
		{"already has shared", []string{"gws-shared", "gws-calendar"}, []string{"gws-shared", "gws-calendar"}},
		{"non-gws unchanged", []string{"email", "notes"}, []string{"email", "notes"}},
		{"multiple gws skills", []string{"gws-drive", "gws-docs"}, []string{"gws-shared", "gws-drive", "gws-docs"}},
		{"empty list", []string{}, []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ensureGWSShared(tt.in)
			if strings.Join(got, ",") != strings.Join(tt.want, ",") {
				t.Errorf("ensureGWSShared(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
