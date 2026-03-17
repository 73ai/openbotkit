package skills

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/73ai/openbotkit/config"
)

func TestParseScopeMap(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   map[string]string
	}{
		{
			name:   "one scope per line",
			output: "https://www.googleapis.com/auth/calendar\nhttps://www.googleapis.com/auth/drive.readonly\n",
			want:   map[string]string{"calendar": "readwrite", "drive": "readonly"},
		},
		{
			name:   "comma separated on one line",
			output: `"scopes": "https://www.googleapis.com/auth/calendar.readonly, https://www.googleapis.com/auth/drive"`,
			want:   map[string]string{"calendar": "readonly", "drive": "readwrite"},
		},
		{
			name:   "readonly not overwritten by broad match",
			output: "https://www.googleapis.com/auth/calendar.readonly",
			want:   map[string]string{"calendar": "readonly"},
		},
		{
			name:   "readwrite when no readonly present",
			output: "https://www.googleapis.com/auth/calendar",
			want:   map[string]string{"calendar": "readwrite"},
		},
		{
			name:   "empty output",
			output: "",
			want:   map[string]string{},
		},
		{
			name:   "multiple services mixed",
			output: "googleapis.com/auth/calendar.readonly\ngoogleapis.com/auth/tasks\ngoogleapis.com/auth/spreadsheets.readonly",
			want:   map[string]string{"calendar": "readonly", "tasks": "readwrite", "sheets": "readonly"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseScopeMap(tt.output)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d entries, want %d: %v", len(got), len(tt.want), got)
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("scope %q: got %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

func TestGWSServiceFromSkillName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"gws-calendar", "calendar"},
		{"gws-calendar-agenda", "calendar"},
		{"gws-calendar-insert", "calendar"},
		{"gws-drive", "drive"},
		{"gws-drive-upload", "drive"},
		{"gws-shared", "shared"},
		{"gws-sheets-append", "sheets"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gwsServiceFromSkillName(tt.name)
			if got != tt.want {
				t.Errorf("gwsServiceFromSkillName(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestInstallBuiltinSkillsNoAuth(t *testing.T) {
	t.Setenv("OBK_CONFIG_DIR", t.TempDir())

	cfg := config.Default()

	result, err := Install(cfg)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	// All built-in skills should be installed regardless of auth state.
	// Auth checks happen at execution time via progressive consent.
	for _, name := range []string{
		"history-read", "memory-save", "web-search", "web-fetch",
		"email-read", "email-send", "whatsapp-read", "whatsapp-send",
		"applenotes-read", "contacts-search",
	} {
		if !slices.Contains(result.Installed, name) {
			t.Errorf("%s should be installed (built-in skills are always installed)", name)
		}
	}

	// Verify SKILL.md was written for history-read.
	tmp := config.Dir()
	content, err := os.ReadFile(filepath.Join(tmp, "skills", "history-read", "SKILL.md"))
	if err != nil {
		t.Fatalf("read history-read SKILL.md: %v", err)
	}
	if len(content) == 0 {
		t.Error("history-read SKILL.md is empty")
	}

	// Verify REFERENCE.md was written alongside SKILL.md.
	refContent, err := os.ReadFile(filepath.Join(tmp, "skills", "history-read", "REFERENCE.md"))
	if err != nil {
		t.Fatalf("read history-read REFERENCE.md: %v", err)
	}
	if len(refContent) == 0 {
		t.Error("history-read REFERENCE.md is empty")
	}

	// Verify schema.sql was written alongside SKILL.md.
	schemaContent, err := os.ReadFile(filepath.Join(tmp, "skills", "history-read", "schema.sql"))
	if err != nil {
		t.Fatalf("read history-read schema.sql: %v", err)
	}
	if len(schemaContent) == 0 {
		t.Error("history-read schema.sql is empty")
	}

	// Verify memory-save SKILL.md and REFERENCE.md were written.
	memorySaveContent, err := os.ReadFile(filepath.Join(tmp, "skills", "memory-save", "SKILL.md"))
	if err != nil {
		t.Fatalf("read memory-save SKILL.md: %v", err)
	}
	if len(memorySaveContent) == 0 {
		t.Error("memory-save SKILL.md is empty")
	}
	memorySaveRef, err := os.ReadFile(filepath.Join(tmp, "skills", "memory-save", "REFERENCE.md"))
	if err != nil {
		t.Fatalf("read memory-save REFERENCE.md: %v", err)
	}
	if len(memorySaveRef) == 0 {
		t.Error("memory-save REFERENCE.md is empty")
	}

	// Verify web-search SKILL.md and REFERENCE.md were written.
	wsContent, err := os.ReadFile(filepath.Join(tmp, "skills", "web-search", "SKILL.md"))
	if err != nil {
		t.Fatalf("read web-search SKILL.md: %v", err)
	}
	if len(wsContent) == 0 {
		t.Error("web-search SKILL.md is empty")
	}
	wsRef, err := os.ReadFile(filepath.Join(tmp, "skills", "web-search", "REFERENCE.md"))
	if err != nil {
		t.Fatalf("read web-search REFERENCE.md: %v", err)
	}
	if len(wsRef) == 0 {
		t.Error("web-search REFERENCE.md is empty")
	}

	// Verify web-fetch SKILL.md and REFERENCE.md were written.
	wfContent, err := os.ReadFile(filepath.Join(tmp, "skills", "web-fetch", "SKILL.md"))
	if err != nil {
		t.Fatalf("read web-fetch SKILL.md: %v", err)
	}
	if len(wfContent) == 0 {
		t.Error("web-fetch SKILL.md is empty")
	}
	wfRef, err := os.ReadFile(filepath.Join(tmp, "skills", "web-fetch", "REFERENCE.md"))
	if err != nil {
		t.Fatalf("read web-fetch REFERENCE.md: %v", err)
	}
	if len(wfRef) == 0 {
		t.Error("web-fetch REFERENCE.md is empty")
	}

	// Verify manifest was written.
	m, err := LoadManifest()
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	if _, ok := m.Skills["history-read"]; !ok {
		t.Error("history-read not in manifest")
	}
	if _, ok := m.Skills["memory-save"]; !ok {
		t.Error("memory-save not in manifest")
	}
	if _, ok := m.Skills["web-search"]; !ok {
		t.Error("web-search not in manifest")
	}
	if _, ok := m.Skills["web-fetch"]; !ok {
		t.Error("web-fetch not in manifest")
	}
}

func TestInstallWithGmailReadonly(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", tmp)

	cfg := config.Default()

	result, err := Install(cfg)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	// All built-in skills installed regardless of auth.
	if !slices.Contains(result.Installed, "email-read") {
		t.Error("email-read should be installed")
	}
	if !slices.Contains(result.Installed, "email-send") {
		t.Error("email-send should be installed")
	}
	if !slices.Contains(result.Installed, "history-read") {
		t.Error("history-read should be installed")
	}
}

func TestInstallWithGmailModify(t *testing.T) {
	t.Setenv("OBK_CONFIG_DIR", t.TempDir())

	cfg := config.Default()

	result, err := Install(cfg)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	// Both email skills always installed, auth checked at execution time.
	if !slices.Contains(result.Installed, "email-read") {
		t.Error("email-read should be installed")
	}
	if !slices.Contains(result.Installed, "email-send") {
		t.Error("email-send should be installed")
	}
}

func TestInstallAlwaysIncludesWhatsApp(t *testing.T) {
	t.Setenv("OBK_CONFIG_DIR", t.TempDir())

	cfg := config.Default()

	result, err := Install(cfg)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	if !slices.Contains(result.Installed, "whatsapp-read") {
		t.Error("whatsapp-read should be installed (always available)")
	}
	if !slices.Contains(result.Installed, "whatsapp-send") {
		t.Error("whatsapp-send should be installed (always available)")
	}
}

func TestInstallAlwaysIncludesAppleNotes(t *testing.T) {
	t.Setenv("OBK_CONFIG_DIR", t.TempDir())

	cfg := config.Default()

	result, err := Install(cfg)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	if !slices.Contains(result.Installed, "applenotes-read") {
		t.Error("applenotes-read should be installed (always available)")
	}
}

func TestInstallKeepsSkillsAfterAuthRevoked(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", tmp)

	cfg := config.Default()

	result1, err := Install(cfg)
	if err != nil {
		t.Fatalf("first Install: %v", err)
	}
	if !slices.Contains(result1.Installed, "whatsapp-read") {
		t.Fatal("whatsapp-read should be installed in first run")
	}

	// Re-install — built-in skills stay regardless of auth state.
	result2, err := Install(cfg)
	if err != nil {
		t.Fatalf("second Install: %v", err)
	}

	if !slices.Contains(result2.Installed, "whatsapp-read") {
		t.Error("whatsapp-read should remain installed (auth checked at execution time)")
	}
	if len(result2.Removed) != 0 {
		t.Errorf("no built-in skills should be removed, got: %v", result2.Removed)
	}
}

func TestSplitSkillFile(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantSlim string
		wantRef  string
	}{
		{
			name:     "no frontmatter",
			input:    "Just some content",
			wantSlim: "Just some content",
			wantRef:  "",
		},
		{
			name:     "frontmatter only",
			input:    "---\nname: test\n---\n",
			wantSlim: "---\nname: test\n---\n",
			wantRef:  "",
		},
		{
			name:     "single paragraph body",
			input:    "---\nname: test\n---\n\nJust a summary line.",
			wantSlim: "---\nname: test\n---\n\nJust a summary line.",
			wantRef:  "",
		},
		{
			name:  "multi paragraph body",
			input: "---\nname: test\ndescription: Test skill\n---\n\nFirst paragraph summary.\n\n## Commands\n\n```bash\nsome command\n```\n\n## Examples\n\nMore content here.",
			wantSlim: "---\nname: test\ndescription: Test skill\n---\n\nFirst paragraph summary.\n\n" +
				"Read the REFERENCE.md in this skill's directory for full instructions.\n",
			wantRef: "## Commands\n\n```bash\nsome command\n```\n\n## Examples\n\nMore content here.\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slim, ref := splitSkillFile([]byte(tt.input))
			if string(slim) != tt.wantSlim {
				t.Errorf("slim:\ngot:  %q\nwant: %q", string(slim), tt.wantSlim)
			}
			if string(ref) != tt.wantRef {
				if tt.wantRef == "" && ref == nil {
					return // nil and "" are equivalent for empty ref
				}
				t.Errorf("ref:\ngot:  %q\nwant: %q", string(ref), tt.wantRef)
			}
		})
	}
}

func TestInstallIdempotent(t *testing.T) {
	t.Setenv("OBK_CONFIG_DIR", t.TempDir())

	cfg := config.Default()

	result1, err := Install(cfg)
	if err != nil {
		t.Fatalf("first Install: %v", err)
	}

	result2, err := Install(cfg)
	if err != nil {
		t.Fatalf("second Install: %v", err)
	}

	// Same skills should be installed both times.
	slices.Sort(result1.Installed)
	slices.Sort(result2.Installed)
	if len(result1.Installed) != len(result2.Installed) {
		t.Fatalf("installed count changed: %d -> %d", len(result1.Installed), len(result2.Installed))
	}
	for i := range result1.Installed {
		if result1.Installed[i] != result2.Installed[i] {
			t.Errorf("installed[%d] changed: %q -> %q", i, result1.Installed[i], result2.Installed[i])
		}
	}

	// No removals on second run.
	if len(result2.Removed) != 0 {
		t.Errorf("second run removed %d skills, want 0", len(result2.Removed))
	}
}
