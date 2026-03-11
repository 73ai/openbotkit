package skills

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/priyanshujain/openbotkit/config"
	"github.com/priyanshujain/openbotkit/oauth/google"
	"golang.org/x/oauth2"
)

func TestIsSkillEligible(t *testing.T) {
	tests := []struct {
		name          string
		meta          SkillMeta
		grantedGoogle map[string]bool
		sourceAuthed  map[string]bool
		want          bool
	}{
		{
			name: "no requirements",
			meta: SkillMeta{},
			want: true,
		},
		{
			name:         "whatsapp required and authed",
			meta:         SkillMeta{RequiresAuth: "whatsapp"},
			sourceAuthed: map[string]bool{"whatsapp": true},
			want:         true,
		},
		{
			name: "whatsapp required but not authed",
			meta: SkillMeta{RequiresAuth: "whatsapp"},
			want: false,
		},
		{
			name:         "applenotes required and linked",
			meta:         SkillMeta{RequiresAuth: "applenotes"},
			sourceAuthed: map[string]bool{"applenotes": true},
			want:         true,
		},
		{
			name: "applenotes required but not linked",
			meta: SkillMeta{RequiresAuth: "applenotes"},
			want: false,
		},
		{
			name: "gmail readonly required and granted",
			meta: SkillMeta{Scopes: []string{"https://www.googleapis.com/auth/gmail.readonly"}},
			grantedGoogle: map[string]bool{
				"https://www.googleapis.com/auth/gmail.readonly": true,
			},
			want: true,
		},
		{
			name:          "gmail readonly required but not granted",
			meta:          SkillMeta{Scopes: []string{"https://www.googleapis.com/auth/gmail.readonly"}},
			grantedGoogle: map[string]bool{},
			want:          false,
		},
		{
			name: "gmail readonly satisfied by modify",
			meta: SkillMeta{Scopes: []string{"https://www.googleapis.com/auth/gmail.readonly"}},
			grantedGoogle: map[string]bool{
				"https://www.googleapis.com/auth/gmail.modify": true,
			},
			want: true,
		},
		{
			name: "gmail modify required and granted",
			meta: SkillMeta{Scopes: []string{"https://www.googleapis.com/auth/gmail.modify"}, Write: true},
			grantedGoogle: map[string]bool{
				"https://www.googleapis.com/auth/gmail.modify": true,
			},
			want: true,
		},
		{
			name: "gmail modify required but only readonly granted",
			meta: SkillMeta{Scopes: []string{"https://www.googleapis.com/auth/gmail.modify"}, Write: true},
			grantedGoogle: map[string]bool{
				"https://www.googleapis.com/auth/gmail.readonly": true,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			granted := tt.grantedGoogle
			if granted == nil {
				granted = map[string]bool{}
			}
			authed := tt.sourceAuthed
			if authed == nil {
				authed = map[string]bool{}
			}
			got := isSkillEligible(tt.meta, granted, authed)
			if got != tt.want {
				t.Errorf("isSkillEligible() = %v, want %v", got, tt.want)
			}
		})
	}
}

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

	// With no auth, history-read, memory-save, web-search, web-fetch should be installed.
	if !slices.Contains(result.Installed, "history-read") {
		t.Error("history-read should be installed (no auth required)")
	}
	if !slices.Contains(result.Installed, "memory-save") {
		t.Error("memory-save should be installed (no auth required)")
	}
	if !slices.Contains(result.Installed, "web-search") {
		t.Error("web-search should be installed (no auth required)")
	}
	if !slices.Contains(result.Installed, "web-fetch") {
		t.Error("web-fetch should be installed (no auth required)")
	}
	if slices.Contains(result.Installed, "email-read") {
		t.Error("email-read should NOT be installed (no gmail auth)")
	}
	if slices.Contains(result.Installed, "whatsapp-read") {
		t.Error("whatsapp-read should NOT be installed (no whatsapp auth)")
	}
	if slices.Contains(result.Installed, "applenotes-read") {
		t.Error("applenotes-read should NOT be installed (not linked)")
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

	// Create a token store with gmail.readonly scope.
	providerDir := filepath.Join(tmp, "providers", "google")
	if err := os.MkdirAll(providerDir, 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	tokenDB := filepath.Join(providerDir, "tokens.db")

	store, err := google.NewTokenStore(tokenDB)
	if err != nil {
		t.Fatalf("create token store: %v", err)
	}
	tok := &oauth2.Token{RefreshToken: "test-refresh", AccessToken: "test-access"}
	if err := store.SaveToken("user@gmail.com", tok, []string{
		"https://www.googleapis.com/auth/gmail.readonly",
	}); err != nil {
		t.Fatalf("save token: %v", err)
	}
	store.Close()

	cfg := config.Default()

	result, err := Install(cfg)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	if !slices.Contains(result.Installed, "email-read") {
		t.Error("email-read should be installed (gmail.readonly granted)")
	}
	if !slices.Contains(result.Installed, "history-read") {
		t.Error("history-read should be installed")
	}
	if slices.Contains(result.Installed, "email-send") {
		t.Error("email-send should NOT be installed (only readonly granted)")
	}
}

func TestInstallWithGmailModify(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", tmp)

	providerDir := filepath.Join(tmp, "providers", "google")
	if err := os.MkdirAll(providerDir, 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	tokenDB := filepath.Join(providerDir, "tokens.db")

	store, err := google.NewTokenStore(tokenDB)
	if err != nil {
		t.Fatalf("create token store: %v", err)
	}
	tok := &oauth2.Token{RefreshToken: "test-refresh", AccessToken: "test-access"}
	if err := store.SaveToken("user@gmail.com", tok, []string{
		"https://www.googleapis.com/auth/gmail.modify",
	}); err != nil {
		t.Fatalf("save token: %v", err)
	}
	store.Close()

	cfg := config.Default()

	result, err := Install(cfg)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	if !slices.Contains(result.Installed, "email-read") {
		t.Error("email-read should be installed (gmail.modify implies readonly)")
	}
	if !slices.Contains(result.Installed, "email-send") {
		t.Error("email-send should be installed (gmail.modify granted)")
	}
}

func TestInstallWithWhatsApp(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", tmp)

	// Create a fake WhatsApp session file.
	waDir := filepath.Join(tmp, "whatsapp")
	if err := os.MkdirAll(waDir, 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(waDir, "session.db"), []byte("fake-session-data"), 0600); err != nil {
		t.Fatalf("write session: %v", err)
	}

	cfg := config.Default()

	result, err := Install(cfg)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	if !slices.Contains(result.Installed, "whatsapp-read") {
		t.Error("whatsapp-read should be installed (session exists)")
	}
	if !slices.Contains(result.Installed, "whatsapp-send") {
		t.Error("whatsapp-send should be installed (session exists)")
	}
}

func TestInstallWithAppleNotes(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", tmp)

	// Simulate linked Apple Notes.
	if err := config.LinkSource("applenotes"); err != nil {
		t.Fatalf("link source: %v", err)
	}

	cfg := config.Default()

	result, err := Install(cfg)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	if !slices.Contains(result.Installed, "applenotes-read") {
		t.Error("applenotes-read should be installed (source linked)")
	}
}

func TestInstallRemovesRevokedSkills(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", tmp)

	// First install with WhatsApp auth.
	waDir := filepath.Join(tmp, "whatsapp")
	if err := os.MkdirAll(waDir, 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	sessionPath := filepath.Join(waDir, "session.db")
	if err := os.WriteFile(sessionPath, []byte("fake-session-data"), 0600); err != nil {
		t.Fatalf("write session: %v", err)
	}

	cfg := config.Default()

	result1, err := Install(cfg)
	if err != nil {
		t.Fatalf("first Install: %v", err)
	}
	if !slices.Contains(result1.Installed, "whatsapp-read") {
		t.Fatal("whatsapp-read should be installed in first run")
	}

	// Remove WhatsApp session (simulates revocation).
	os.Remove(sessionPath)

	// Re-install.
	result2, err := Install(cfg)
	if err != nil {
		t.Fatalf("second Install: %v", err)
	}

	if slices.Contains(result2.Installed, "whatsapp-read") {
		t.Error("whatsapp-read should NOT be installed after session removed")
	}
	if !slices.Contains(result2.Removed, "whatsapp-read") {
		t.Error("whatsapp-read should be in removed list")
	}

	// Verify file was actually removed.
	skillDir := filepath.Join(tmp, "skills", "whatsapp-read")
	if _, err := os.Stat(skillDir); !os.IsNotExist(err) {
		t.Error("whatsapp-read directory should have been removed")
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
