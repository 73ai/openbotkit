package tools

import (
	"strings"
	"testing"
)

func TestTruncateHead_UnderLimit(t *testing.T) {
	input := "line1\nline2\nline3"
	got := TruncateHead(input, 5)
	if got != input {
		t.Errorf("expected passthrough, got %q", got)
	}
}

func TestTruncateHead_ExactLimit(t *testing.T) {
	input := "line1\nline2\nline3"
	got := TruncateHead(input, 3)
	if got != input {
		t.Errorf("expected passthrough at exact limit, got %q", got)
	}
}

func TestTruncateHead_OverLimit(t *testing.T) {
	lines := make([]string, 100)
	for i := range lines {
		lines[i] = "line"
	}
	input := strings.Join(lines, "\n")
	got := TruncateHead(input, 10)

	gotLines := strings.Split(got, "\n")
	// 10 kept lines + 1 marker line = 11
	if len(gotLines) != 11 {
		t.Errorf("got %d lines, want 11", len(gotLines))
	}
	if !strings.Contains(got, "[truncated: showing 10 of 100 lines]") {
		t.Errorf("missing truncation marker in %q", got)
	}
}

func TestTruncateTail_UnderLimit(t *testing.T) {
	input := "line1\nline2"
	got := TruncateTail(input, 5)
	if got != input {
		t.Errorf("expected passthrough, got %q", got)
	}
}

func TestTruncateTail_OverLimit(t *testing.T) {
	lines := make([]string, 100)
	for i := range lines {
		lines[i] = "line"
	}
	input := strings.Join(lines, "\n")
	got := TruncateTail(input, 10)

	if !strings.Contains(got, "[truncated: showing 10 of 100 lines]") {
		t.Errorf("missing truncation marker in %q", got)
	}
	// Marker is first line, then 10 kept lines = 11 total.
	gotLines := strings.Split(got, "\n")
	if len(gotLines) != 11 {
		t.Errorf("got %d lines, want 11", len(gotLines))
	}
}

func TestTruncateHeadTail_UnderLimit(t *testing.T) {
	input := "a\nb\nc\nd\ne"
	got := TruncateHeadTail(input, 3, 3)
	if got != input {
		t.Errorf("expected passthrough (5 <= 3+3), got %q", got)
	}
}

func TestTruncateHeadTail_OverLimit(t *testing.T) {
	lines := make([]string, 100)
	for i := range lines {
		lines[i] = "line"
	}
	input := strings.Join(lines, "\n")
	got := TruncateHeadTail(input, 5, 5)

	if !strings.Contains(got, "[truncated: showing 5+5 of 100 lines]") {
		t.Errorf("missing truncation marker in %q", got)
	}
}

func TestTruncateBytes_UnderLimit(t *testing.T) {
	input := "hello"
	got := TruncateBytes(input, 1024)
	if got != input {
		t.Errorf("expected passthrough, got %q", got)
	}
}

func TestTruncateBytes_OverLimit(t *testing.T) {
	input := strings.Repeat("a", 10000)
	got := TruncateBytes(input, 100)
	if len(got) > 200 { // 100 bytes + marker
		t.Errorf("output too long: %d bytes", len(got))
	}
	if !strings.Contains(got, "[truncated:") {
		t.Errorf("missing truncation marker")
	}
}

func TestTruncate_EmptyString(t *testing.T) {
	if got := TruncateHead("", 10); got != "" {
		t.Errorf("TruncateHead empty = %q", got)
	}
	if got := TruncateTail("", 10); got != "" {
		t.Errorf("TruncateTail empty = %q", got)
	}
	if got := TruncateHeadTail("", 5, 5); got != "" {
		t.Errorf("TruncateHeadTail empty = %q", got)
	}
	if got := TruncateBytes("", 100); got != "" {
		t.Errorf("TruncateBytes empty = %q", got)
	}
}

func TestTruncate_NoNewlines(t *testing.T) {
	// Single huge line — line-based truncation passes through, byte truncation kicks in.
	input := strings.Repeat("x", 100000)
	got := TruncateHead(input, 10)
	if got != input {
		t.Error("single line should pass through TruncateHead")
	}
	got = TruncateBytes(input, 1024)
	if !strings.Contains(got, "[truncated:") {
		t.Error("TruncateBytes should truncate single huge line")
	}
}

func TestTruncateBytes_BinaryContent(t *testing.T) {
	// Non-UTF8 bytes.
	input := string([]byte{0xff, 0xfe, 0x80, 0x81, 0x82, 0x83, 0x84, 0x85})
	got := TruncateBytes(input, 4)
	if !strings.Contains(got, "[truncated:") {
		t.Error("expected truncation marker for binary content")
	}
}

func TestTruncateBytes_MultiByte(t *testing.T) {
	// "café" = 63 61 66 c3 a9 (5 bytes, 4 runes).
	// Cutting at byte 4 lands mid-rune (inside the é).
	// TruncateBytes must back up to produce valid UTF-8.
	input := "café"
	got := TruncateBytes(input, 4)
	// Should contain "caf" (3 bytes) but NOT the broken é.
	if strings.Contains(got, "é") {
		t.Error("should not contain full é when cut at byte 4")
	}
	if !strings.HasPrefix(got, "caf") {
		t.Errorf("expected prefix 'caf', got %q", got)
	}

	// 3-byte rune: "日本語" = e6 97 a5 | e6 9c ac | e8 aa 9e (9 bytes).
	// Cut at 4 should keep "日" (3 bytes) and drop partial second rune.
	input2 := "日本語"
	got2 := TruncateBytes(input2, 4)
	if !strings.HasPrefix(got2, "日") {
		t.Errorf("expected prefix '日', got %q", got2)
	}
	if strings.HasPrefix(got2, "日本") {
		t.Error("should not contain second CJK character when cut at byte 4")
	}

	// 4-byte rune: emoji "😀" = f0 9f 98 80.
	// Cut at 3 should produce empty prefix (can't keep partial emoji).
	input3 := "😀hello"
	got3 := TruncateBytes(input3, 3)
	if strings.Contains(got3, "😀") {
		t.Error("should not contain emoji when cut at byte 3")
	}
}
