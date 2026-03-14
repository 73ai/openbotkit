package tghtml

import (
	"strings"
	"testing"
)

func TestConvert_Bold(t *testing.T) {
	got := Convert("**hello**")
	if !strings.Contains(got, "<b>hello</b>") {
		t.Errorf("expected <b>hello</b>, got %q", got)
	}
}

func TestConvert_Italic(t *testing.T) {
	got := Convert("*hello*")
	if !strings.Contains(got, "<i>hello</i>") {
		t.Errorf("expected <i>hello</i>, got %q", got)
	}
}

func TestConvert_Code(t *testing.T) {
	got := Convert("`code`")
	if !strings.Contains(got, "<code>code</code>") {
		t.Errorf("expected <code>code</code>, got %q", got)
	}
}

func TestConvert_CodeBlock(t *testing.T) {
	got := Convert("```go\nfmt.Println()\n```")
	if !strings.Contains(got, "<pre>") {
		t.Errorf("expected <pre> block, got %q", got)
	}
}

func TestConvert_Link(t *testing.T) {
	got := Convert("[click](https://example.com)")
	if !strings.Contains(got, `<a href="https://example.com">click</a>`) {
		t.Errorf("expected link, got %q", got)
	}
}

func TestConvert_HTMLEscaped(t *testing.T) {
	got := Convert("a < b & c > d")
	if !strings.Contains(got, "&lt;") || !strings.Contains(got, "&amp;") || !strings.Contains(got, "&gt;") {
		t.Errorf("expected HTML escaping, got %q", got)
	}
}

func TestConvert_PlainText(t *testing.T) {
	got := Convert("just plain text")
	if !strings.Contains(got, "just plain text") {
		t.Errorf("expected plain text preserved, got %q", got)
	}
}

func TestConvert_EmptyInput(t *testing.T) {
	got := Convert("")
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestConvert_Heading(t *testing.T) {
	got := Convert("# My Heading")
	if !strings.Contains(got, "<b>My Heading</b>") {
		t.Errorf("expected <b>My Heading</b>, got %q", got)
	}
}

func TestConvert_Blockquote(t *testing.T) {
	got := Convert("> quoted text")
	if !strings.Contains(got, "<blockquote>") || !strings.Contains(got, "quoted text") {
		t.Errorf("expected blockquote with text, got %q", got)
	}
}

func TestConvert_UnorderedList(t *testing.T) {
	got := Convert("- first\n- second")
	if !strings.Contains(got, "• first") || !strings.Contains(got, "• second") {
		t.Errorf("expected bullet items, got %q", got)
	}
}

func TestConvert_ThematicBreak(t *testing.T) {
	got := Convert("above\n\n---\n\nbelow")
	if strings.Count(got, "---") < 1 {
		t.Errorf("expected --- thematic break, got %q", got)
	}
}

func TestConvert_ImageAltTextOnly(t *testing.T) {
	got := Convert("![alt text](https://example.com/img.png)")
	if !strings.Contains(got, "alt text") {
		t.Errorf("expected alt text preserved, got %q", got)
	}
	if strings.Contains(got, "<img") {
		t.Errorf("expected no <img> tag, got %q", got)
	}
}

func TestConvert_AutoLinkEmail(t *testing.T) {
	got := Convert("<user@example.com>")
	if !strings.Contains(got, "mailto:") {
		t.Errorf("expected mailto: link, got %q", got)
	}
	if !strings.Contains(got, "user@example.com") {
		t.Errorf("expected email in output, got %q", got)
	}
}

func TestConvert_CodeBlockNoLanguage(t *testing.T) {
	got := Convert("```\nplain code\n```")
	if !strings.Contains(got, "<pre>") || !strings.Contains(got, "plain code") {
		t.Errorf("expected <pre> with plain code, got %q", got)
	}
	if strings.Contains(got, "language-") {
		t.Errorf("expected no language class, got %q", got)
	}
}

func TestConvert_NestedFormatting(t *testing.T) {
	got := Convert("**bold and *italic***")
	if !strings.Contains(got, "<b>") || !strings.Contains(got, "<i>") {
		t.Errorf("expected nested bold+italic, got %q", got)
	}
}
