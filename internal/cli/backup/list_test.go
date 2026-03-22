package backup

import (
	"strings"
	"testing"
)

func TestFormatSnapshotDate(t *testing.T) {
	// Valid ID with random suffix.
	got := formatSnapshotDate("20260321T150405Z-abcd1234")
	if !strings.Contains(got, "2026") || !strings.Contains(got, "03") {
		t.Errorf("expected formatted date, got %q", got)
	}

	// Invalid ID returns blank padding.
	got = formatSnapshotDate("invalid")
	if strings.TrimSpace(got) != "" {
		t.Errorf("expected blank for invalid ID, got %q", got)
	}

	// Old format ID without suffix (backwards compat).
	got = formatSnapshotDate("20260321T150405Z")
	if !strings.Contains(got, "2026") {
		t.Errorf("expected formatted date for old format, got %q", got)
	}
}
