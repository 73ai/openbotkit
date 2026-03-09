package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateIndex(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", tmp)

	// Create skill directories with SKILL.md files.
	skills := map[string]string{
		"email-read": `---
name: email-read
description: Search and read emails from Gmail inbox
allowed-tools: Bash(sqlite3 *)
---

## Database
Path: ~/.obk/gmail/data.db
`,
		"whatsapp-read": `---
name: whatsapp-read
description: Search WhatsApp messages and browse chats
allowed-tools: Bash(sqlite3 *)
---

## Database
Path: ~/.obk/whatsapp/data.db
`,
	}

	for name, content := range skills {
		dir := filepath.Join(tmp, "skills", name)
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatalf("mkdir %s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	if err := GenerateIndex(); err != nil {
		t.Fatalf("GenerateIndex: %v", err)
	}

	idx, err := LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex: %v", err)
	}

	if len(idx.Skills) != 2 {
		t.Fatalf("got %d skills, want 2", len(idx.Skills))
	}

	found := make(map[string]string)
	for _, s := range idx.Skills {
		found[s.Name] = s.Description
	}

	if found["email-read"] != "Search and read emails from Gmail inbox" {
		t.Errorf("email-read description = %q", found["email-read"])
	}
	if found["whatsapp-read"] != "Search WhatsApp messages and browse chats" {
		t.Errorf("whatsapp-read description = %q", found["whatsapp-read"])
	}
}

func TestGenerateIndexEmpty(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", tmp)

	// Create empty skills directory.
	if err := os.MkdirAll(filepath.Join(tmp, "skills"), 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := GenerateIndex(); err != nil {
		t.Fatalf("GenerateIndex: %v", err)
	}

	idx, err := LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex: %v", err)
	}

	if len(idx.Skills) != 0 {
		t.Errorf("got %d skills, want 0", len(idx.Skills))
	}
}

func TestGenerateIndexBadFrontmatter(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", tmp)

	skillsDir := filepath.Join(tmp, "skills")

	// Good skill.
	goodDir := filepath.Join(skillsDir, "good-skill")
	if err := os.MkdirAll(goodDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(goodDir, "SKILL.md"), []byte(`---
name: good-skill
description: A valid skill
---
`), 0600); err != nil {
		t.Fatal(err)
	}

	// Bad skill (invalid YAML).
	badDir := filepath.Join(skillsDir, "bad-skill")
	if err := os.MkdirAll(badDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(badDir, "SKILL.md"), []byte(`---
: invalid yaml [
---
`), 0600); err != nil {
		t.Fatal(err)
	}

	if err := GenerateIndex(); err != nil {
		t.Fatalf("GenerateIndex: %v", err)
	}

	idx, err := LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex: %v", err)
	}

	// Only the good skill should be indexed.
	if len(idx.Skills) != 1 {
		t.Fatalf("got %d skills, want 1", len(idx.Skills))
	}
	if idx.Skills[0].Name != "good-skill" {
		t.Errorf("got name %q, want good-skill", idx.Skills[0].Name)
	}
}

func TestLoadIndex(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", tmp)

	// Round-trip test.
	original := &Index{
		Skills: []IndexEntry{
			{Name: "email-read", Description: "Read emails"},
			{Name: "history-read", Description: "Recall conversations"},
		},
	}

	if err := SaveIndex(original); err != nil {
		t.Fatalf("SaveIndex: %v", err)
	}

	loaded, err := LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex: %v", err)
	}

	if len(loaded.Skills) != len(original.Skills) {
		t.Fatalf("got %d skills, want %d", len(loaded.Skills), len(original.Skills))
	}
	for i, s := range loaded.Skills {
		if s.Name != original.Skills[i].Name || s.Description != original.Skills[i].Description {
			t.Errorf("skill[%d] = %+v, want %+v", i, s, original.Skills[i])
		}
	}
}
